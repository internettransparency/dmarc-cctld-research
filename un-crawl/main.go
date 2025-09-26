package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	countryHeaderCandidates  = []string{"country", "country name", "countryname"}
	nationalHeaderCandidates = []string{"national portal", "nationalportal", "national government portal"}
	otherHeaderCandidates    = []string{"other portals", "otherportal", "other government portals"}
)

func normalizeHeader(h string) string {
	return strings.ToLower(strings.Join(strings.Fields(h), " "))
}

func headerMatches(h string, candidates []string) bool {
	n := normalizeHeader(h)
	nNoSpace := strings.ReplaceAll(n, " ", "")
	for _, c := range candidates {
		if n == c || nNoSpace == c {
			return true
		}
	}
	return false
}

func main() {
	defaultURL := "https://publicadministration.un.org/egovkb/en-us/Resources/Country-URLs"
	pageURL := flag.String("url", defaultURL, "URL of the UN EGOVKB Country-URLs page")
	userAgent := flag.String("ua", "Mozilla/5.0 (compatible; egovkb-scraper/1.1; +https://example.invalid)", "HTTP User-Agent")
	retries := flag.Int("retries", 4, "Number of fetch retries on transient errors")
	flag.Parse()

	doc, base, err := fetchDoc(*pageURL, *userAgent, *retries)
	if err != nil {
		log.Fatalf("failed to fetch page: %v", err)
	}

	tableSel := findCountryUrlsTable(doc)
	if tableSel == nil {
		log.Fatalf("could not find a table with headers matching [Country, National Portal, Other Portals]")
	}

	countryIdx, nationalIdx, otherIdx, err := locateColumns(tableSel)
	if err != nil {
		log.Fatalf("failed to map expected columns: %v", err)
	}

	outFile, err := os.Create("un_country_gov_portals.csv")
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	w := csv.NewWriter(outFile)
	defer w.Flush()

	if err := w.Write([]string{"country_name", "national_portal", "other_portals"}); err != nil {
		log.Fatalf("csv write header: %v", err)
	}

	rows := tableSel.Find("tbody tr")
	if rows.Length() == 0 {
		rows = tableSel.Find("tr")
	}
	rows = rows.Slice(1, goquery.ToEnd) // <--- skip the first row
	rows.Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() == 0 {
			tds = tr.Find("th,td")
		}
		if tds.Length() == 0 {
			return
		}

		country := strings.TrimSpace(tds.Eq(countryIdx).Text())
		if country == "" {
			return
		}
		nationalPortal := extractFirstURL(tds.Eq(nationalIdx), base)
		otherPortals := extractAllURLs(tds.Eq(otherIdx), base)
		otherJoined := strings.Join(otherPortals, ";")

		if err := w.Write([]string{country, nationalPortal, otherJoined}); err != nil {
			log.Printf("csv write row error for %q: %v", country, err)
		}
	})
}

func fetchDoc(pageURL, ua string, retries int) (*goquery.Document, *url.URL, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid url: %w", err)
	}

	// 1) Try HTTP/2-only first (avoids broken HTTP/1.1 header names).
	doc, _, err := fetchHTTP2(pageURL, ua, retries)
	if err == nil {
		return doc, u, nil
	}
	// If the error is the infamous malformed header / HTTP/1.x transport break or 5xx, fall back to curl/wget.
	if !isHeaderParseOrServerError(err) {
		// Not the known error type, but still try curl as a robust fallback.
	}

	// 2) Fallback: use curl (or wget) to get the HTML and parse it.
	html, cerr := fetchWithCurlOrWget(pageURL, ua, retries)
	if cerr != nil {
		return nil, nil, fmt.Errorf("http2 fetch failed: %v; curl/wget fallback failed: %w", err, cerr)
	}
	doc2, perr := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if perr != nil {
		return nil, nil, fmt.Errorf("parse html (curl fallback): %w", perr)
	}
	return doc2, u, nil
}

func fetchHTTP2(pageURL, ua string, retries int) (*goquery.Document, *url.URL, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, nil, err
	}
	tr := &http.Transport{
		// Force ALPN to h2 only; avoids falling back to HTTP/1.1 (where the broken header shows up).
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"h2"},
		},
		ForceAttemptHTTP2: true,
		// Be conservative with connections; this site can be flaky.
		DisableKeepAlives: true,
		MaxIdleConns:      0,
	}
	client := &http.Client{
		Timeout:   45 * time.Second,
		Transport: tr,
	}

	var lastErr error
	for i := 0; i <= retries; i++ {
		req, _ := http.NewRequest("GET", base.String(), nil)
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.8")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				defer resp.Body.Close()
				doc, perr := goquery.NewDocumentFromReader(resp.Body)
				if perr != nil {
					return nil, nil, perr
				}
				return doc, base, nil
			}
			lastErr = fmt.Errorf("bad status %d", resp.StatusCode)
			resp.Body.Close()
		}
		time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
	}
	return nil, nil, lastErr
}

func isHeaderParseOrServerError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "malformed MIME header line") ||
		strings.Contains(s, "HTTP/1.x transport connection broken") ||
		strings.Contains(s, "502") || strings.Contains(s, "503") || strings.Contains(s, "504")
}

func fetchWithCurlOrWget(pageURL, ua string, retries int) ([]byte, error) {
	var out []byte
	var err error

	curlPath, _ := exec.LookPath("curl")
	wgetPath, _ := exec.LookPath("wget")

	for i := 0; i <= retries; i++ {
		switch {
		case curlPath != "":
			out, err = exec.Command(
				curlPath,
				"-sS", "-L", "--compressed",
				"-H", "User-Agent: "+ua,
				"-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
				"-H", "Accept-Language: en-US,en;q=0.8",
				pageURL,
			).Output()
		case wgetPath != "":
			out, err = exec.Command(
				wgetPath,
				"-q", "-O", "-",
				"--header", "User-Agent: "+ua,
				"--header", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
				"--header", "Accept-Language: en-US,en;q=0.8",
				pageURL,
			).Output()
		default:
			return nil, errors.New("neither curl nor wget found in PATH")
		}

		if err == nil && len(out) > 0 {
			return out, nil
		}
		time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
	}
	if err == nil {
		err = errors.New("failed to fetch with curl/wget")
	}
	return nil, err
}

func findCountryUrlsTable(doc *goquery.Document) *goquery.Selection {
	var chosen *goquery.Selection
	doc.Find("table").EachWithBreak(func(_ int, t *goquery.Selection) bool {
		countryIdx, nationalIdx, otherIdx, err := locateColumns(t)
		if err == nil && countryIdx >= 0 && nationalIdx >= 0 && otherIdx >= 0 {
			chosen = t
			return false
		}
		return true
	})
	return chosen
}

func locateColumns(table *goquery.Selection) (countryIdx, nationalIdx, otherIdx int, err error) {
	countryIdx, nationalIdx, otherIdx = -1, -1, -1
	headerCells := table.Find("thead th")
	if headerCells.Length() == 0 {
		headerCells = table.Find("tr").First().Find("th,td")
	}
	headerCells.Each(func(i int, th *goquery.Selection) {
		text := th.Text()
		switch {
		case headerMatches(text, countryHeaderCandidates):
			countryIdx = i
		case headerMatches(text, nationalHeaderCandidates):
			nationalIdx = i
		case headerMatches(text, otherHeaderCandidates):
			otherIdx = i
		}
	})
	if countryIdx == -1 || nationalIdx == -1 || otherIdx == -1 {
		return -1, -1, -1, fmt.Errorf("expected headers not found (got country=%d national=%d other=%d)", countryIdx, nationalIdx, otherIdx)
	}
	return countryIdx, nationalIdx, otherIdx, nil
}

func extractFirstURL(td *goquery.Selection, base *url.URL) string {
	links := extractAllURLs(td, base)
	if len(links) > 0 {
		return links[0]
	}
	return ""
}

func extractAllURLs(td *goquery.Selection, base *url.URL) []string {
	var out []string
	td.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		h := strings.TrimSpace(href)
		if h == "" {
			return
		}
		u, err := base.Parse(h)
		if err != nil {
			return
		}
		if u.Scheme == "http" || u.Scheme == "https" {
			out = append(out, u.String())
		}
	})
	// de-dup, preserve order
	seen := map[string]struct{}{}
	dedup := make([]string, 0, len(out))
	for _, v := range out {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			dedup = append(dedup, v)
		}
	}
	return dedup
}
