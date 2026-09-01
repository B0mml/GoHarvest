package scraper

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Bommel48/go-scraper-notifier/pkg/parser"
	"github.com/gocolly/colly/v2"
)

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(10 * time.Second)

	// Set Steam age verification
	c.SetCookies("https://store.steampowered.com", []*http.Cookie{
		{Name: "birthtime", Value: "283993201", Domain: "store.steampowered.com", Path: "/"},
		{Name: "mature_content", Value: "1", Domain: "store.steampowered.com", Path: "/"},
		{Name: "wants_mature_content", Value: "1", Domain: "store.steampowered.com", Path: "/"},
	})

	return c
}

// scrape visits a URL and extracts the raw price string automatically.
func scrape(c *colly.Collector, url string) (string, error) {
	collector := c.Clone()

	var rawPrice string

	// Helper to extract value of content or inner text
	extract := func(e *colly.HTMLElement) string {
		if content := e.Attr("content"); content != "" {
			return content
		}
		return strings.TrimSpace(e.Text)
	}

	// Automatic price detection: standard metadata tags and common price selectors
	selectors := []string{
		`meta[property="product:price:amount"]`,
		`meta[property="og:price:amount"]`,
		`[itemprop="price"]`,
		`meta[itemprop="price"]`,
		`.product-detail-price`,
		`div._pbox .preis`,
		`.preis`,
		`[class*="preis"]`,
		`.price`,
		`[class*="price"]`,
	}
	for _, sel := range selectors {
		collector.OnHTML(sel, func(e *colly.HTMLElement) {
			if rawPrice == "" {
				rawPrice = extract(e)
			}
		})
	}

	var visitErr error
	collector.OnError(func(r *colly.Response, err error) {
		visitErr = err
	})

	if err := collector.Visit(url); err != nil {
		return "", fmt.Errorf("failed to visit %s: %w", url, err)
	}
	if visitErr != nil {
		return "", fmt.Errorf("request failed for %s: %w", url, visitErr)
	}
	if rawPrice == "" {
		return "", fmt.Errorf("could not find price on %s", url)
	}

	return rawPrice, nil
}

func ScrapePrice(url string) (float64, error) {
	c := newCollector()
	priceStr, err := scrape(c, url)
	if err != nil {
		log.Printf("Error scraping url: %s %v", url, err)
		return 0, err
	}

	price, err := parser.CleanPrice(priceStr)
	if err != nil {
		log.Printf("Error converting price to float: %v", err)
		return 0, err
	}

	return price, err
}
