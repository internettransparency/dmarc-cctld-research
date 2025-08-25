package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Country represents a country with its name and URL
type Country struct {
	Name string
	URL  string
}

// TableRow represents a row from the government directory table
type TableRow struct {
	Name            string
	GovdirectoryURL string
	Type            string
	Website         string
}

// CrawlGovDirectory extracts all list items from the main container
func CrawlGovDirectory(url string) ([]Country, error) {
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var countries []Country

	// Find all list items within main.container
	doc.Find("main.container li").Each(func(i int, s *goquery.Selection) {
		// Extract the link within each list item
		link := s.Find("a")
		if link.Length() > 0 {
			countryName := strings.TrimSpace(link.Text())
			countryURL, exists := link.Attr("href")

			if exists && countryName != "" {
				// Make the URL absolute if it's relative
				if !strings.HasPrefix(countryURL, "http") {
					countryURL = "https://www.govdirectory.org" + countryURL
				}

				countries = append(countries, Country{
					Name: countryName,
					URL:  countryURL,
				})
			}
		}
	})

	return countries, nil
}

// ExtractTableDataAndGenerateTSV fetches a country page and generates TSV from the table
func ExtractTableDataAndGenerateTSV(country Country) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Fetch the country page
	resp, err := client.Get(country.URL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page %s: %w", country.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code error for %s: %d %s", country.Name, resp.StatusCode, resp.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML for %s: %w", country.Name, err)
	}

	// Extract table rows
	var rows []TableRow

	// Find the table and iterate through tbody rows
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		row := TableRow{}

		// Extract Name (first column)
		nameCol := s.Find("td").Eq(0)
		row.Name = strings.TrimSpace(nameCol.Text())

		// Extract Govdirectory URL from the link in first column
		if link := nameCol.Find("a").First(); link.Length() > 0 {
			href, exists := link.Attr("href")
			if exists {
				if !strings.HasPrefix(href, "http") {
					href = "https://www.govdirectory.org/" + strings.ToLower(country.Name) + "/" + href
				}
				row.GovdirectoryURL = href
			}
		}

		// Extract Website (second column) - look for the Website link
		websiteCol := s.Find("td").Eq(1)
		websiteCol.Find("a").Each(func(j int, link *goquery.Selection) {
			title, hasTitle := link.Attr("title")
			if hasTitle && title == "Website" {
				if href, exists := link.Attr("href"); exists {
					row.Website = href
					return
				}
			}
		})

		// Extract Type (third column)
		typeCol := s.Find("td").Eq(2)
		row.Type = strings.TrimSpace(typeCol.Text())

		// Add row to list if it has a name
		if row.Name != "" {
			rows = append(rows, row)
		}
	})

	// Generate TSV content exactly as the JavaScript does
	tsv := "\"Name\"\t\"Govdirectory URL\"\t\"Type\"\t\"Website\"\n"

	for _, row := range rows {
		// Format each row with quotes and tabs, exactly matching the JavaScript
		tsvRow := fmt.Sprintf("\"\"%s\"\t\"%s\"\t\"%s\"\t\"%s\"\n",
			row.Name,
			row.GovdirectoryURL,
			row.Type,
			row.Website)
		tsv += tsvRow
	}

	return tsv, nil
}

// DownloadTSV processes a country page and saves the generated TSV
func DownloadTSV(country Country, outputDir string) error {
	fmt.Printf("  → Extracting table data from: %s\n", country.URL)

	// Extract table data and generate TSV
	tsvContent, err := ExtractTableDataAndGenerateTSV(country)
	if err != nil {
		return err
	}

	// Create filename - using the same format as the download button
	filename := country.Name + ".tsv"
	filepath := filepath.Join(outputDir, filename)

	// Write to file
	err = os.WriteFile(filepath, []byte(tsvContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write TSV for %s: %w", country.Name, err)
	}

	// Count rows for statistics (subtract 1 for header)
	rowCount := strings.Count(tsvContent, "\n") - 1

	fmt.Printf("✓ Generated TSV: %s -> %s (%d rows, %d bytes)\n",
		country.Name, filename, rowCount, len(tsvContent))

	return nil
}

// DownloadAllTSVsSequential downloads all TSV files one by one
func DownloadAllTSVsSequential(countries []Country, outputDir string) error {
	// Create output directory if it doesn't exist
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	successCount := 0
	errorCount := 0
	totalRows := 0
	var failedCountries []string

	for i, country := range countries {
		fmt.Printf("\n[%d/%d] Processing %s...\n", i+1, len(countries), country.Name)

		if err := DownloadTSV(country, outputDir); err != nil {
			fmt.Printf("✗ Error: %v\n", err)
			errorCount++
			failedCountries = append(failedCountries, country.Name)
		} else {
			successCount++

			// Read file to count rows for statistics
			filepath := filepath.Join(outputDir, country.Name+".tsv")
			if content, err := os.ReadFile(filepath); err == nil {
				totalRows += strings.Count(string(content), "\n") - 1
			}
		}

		// Be respectful to the server - wait between requests
		if i < len(countries)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Print("\n" + strings.Repeat("=", 50))
	fmt.Printf("\nDownload complete: %d successful, %d errors\n", successCount, errorCount)
	fmt.Printf("Total data rows extracted: %d\n", totalRows)

	if len(failedCountries) > 0 {
		fmt.Println("\nFailed countries:")
		for _, name := range failedCountries {
			fmt.Printf("  - %s\n", name)
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("encountered %d errors during download", errorCount)
	}

	return nil
}

// CombineAllTSVs combines all TSV files into a single master file
func CombineAllTSVs(outputDir string) error {
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	masterFile := filepath.Join(outputDir, "_ALL_COUNTRIES_COMBINED.tsv")
	output, err := os.Create(masterFile)
	if err != nil {
		return fmt.Errorf("failed to create master file: %w", err)
	}
	defer output.Close()

	// Write header with additional Country column
	_, err = output.WriteString("\"Country\"\t\"Name\"\t\"Govdirectory URL\"\t\"Type\"\t\"Website\"\n")
	if err != nil {
		return err
	}

	totalRows := 0
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".tsv") || file.Name() == "_ALL_COUNTRIES_COMBINED.tsv" {
			continue
		}

		filepath := filepath.Join(outputDir, file.Name())
		content, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Printf("Warning: Could not read %s: %v\n", file.Name(), err)
			continue
		}

		// Get country name from filename
		countryName := strings.TrimSuffix(file.Name(), ".tsv")

		// Split into lines and skip header
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if i == 0 || line == "" { // Skip header and empty lines
				continue
			}

			// Add country name as first column
			modifiedLine := fmt.Sprintf("\"%s\"\t%s", countryName, line)
			_, err = output.WriteString(modifiedLine + "\n")
			if err != nil {
				return err
			}
			totalRows++
		}
	}

	fmt.Printf("\n✓ Created combined TSV file: %s\n", masterFile)
	fmt.Printf("  Total rows in combined file: %d\n", totalRows)

	return nil
}

func main() {
	url := "https://www.govdirectory.org/countries/"
	outputDir := "country_tsv_files"

	fmt.Println("GovDirectory.org TSV Extractor")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Step 1: Extract countries from main page
	fmt.Println("\nStep 1: Extracting country list...")
	countries, err := CrawlGovDirectory(url)
	if err != nil {
		log.Fatal("Error crawling website:", err)
	}

	fmt.Printf("Found %d countries\n", len(countries))

	// Step 2: Extract table data and generate TSV files
	fmt.Println("\nStep 2: Extracting table data and generating TSV files...")
	fmt.Printf("Output directory: %s\n", outputDir)

	// Sequential downloads (slower but more server-friendly)
	fmt.Println("Using sequential extraction (server-friendly)...")
	err = DownloadAllTSVsSequential(countries, outputDir)

	if err != nil {
		fmt.Printf("\nWarning: %v\n", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("All extraction completed! Check the '%s' directory.\n", outputDir)

	// Print summary of downloaded files
	files, err := os.ReadDir(outputDir)
	if err == nil {
		tsvCount := 0
		var totalSize int64
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".tsv") {
				tsvCount++
				info, err := file.Info()
				if err == nil {
					totalSize += info.Size()
				}
			}
		}
		fmt.Printf("Total TSV files: %d\n", tsvCount)
		fmt.Printf("Total size: %.2f MB\n", float64(totalSize)/(1024*1024))
	}
}
