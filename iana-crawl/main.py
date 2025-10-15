import requests
from bs4 import BeautifulSoup as bs
from urllib.parse import urlparse

# to scrape, you need to also click on each tld link and find out which country it belongs to
url = "https://www.iana.org/domains/root/db"
links = []
resp = requests.get(url)
soup = bs(resp.text,'lxml')
og = soup.find("meta",  property="og:url")
base = urlparse(url)
for link in soup.find_all('td'):
    current_link = link.get('href')
    if current_link.endswith('pdf'):
        if og:
            links.append(og["content"] + current_link)
        else:
            links.append(base.scheme+"://"+base.netloc + current_link)
