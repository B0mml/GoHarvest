package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Bommel48/go-scraper-notifier/pkg/models"
	"github.com/Bommel48/go-scraper-notifier/pkg/parser"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	"github.com/gocolly/colly/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

func publishArticles(ch *amqp.Channel, queueName string, count int) error {
	log.Println("Starting test...")

	for i := 1; i <= count; i++ {
		article := models.Article{
			UserID: (i % 3) + 1,
			Title:  fmt.Sprintf("Test article #%d", i),
			URL:    fmt.Sprintf("https://example.com/test-%d", i),
			Price:  19.99 + float64(i),
		}

		if err := rbmq.Publish(ch, queueName, article); err != nil {
			log.Printf("Error sending article #%d: %v", i, err)
		}

		time.Sleep(1 * time.Millisecond)
	}

	log.Println("Done!")
	return nil
}

func setupColly() *colly.Collector {
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

func parse(raw string) (float64, error) {
	return parser.CleanPrice(raw)
}

func main() {
	// amqpURL := "amqp://guest:guest@rabbitmq:5672/"
	//
	// ch, err := rbmq.Connect(amqpURL, 10, 2*time.Second)
	// if err != nil {
	// 	log.Fatalf("RabbitMQ connection failure: %v", err)
	// }
	// defer ch.Close()
	//
	// // Test
	// if err := publishArticles(ch, rbmq.QueueName, 5000); err != nil {
	// 	log.Fatalf("Publish error: %v", err)
	// }

	c := setupColly()

	urls := []string{
		"https://www.spiele-offensive.de/Spiel/Turbo-Flitzpiepen-2000-1034429.html",
		"https://www.rockshop.de/elektron-digitakt-ii",
		"https://store.steampowered.com/app/1245620/",
	}

	for _, url := range urls {
		raw, err := scrape(c, url)
		if err != nil {
			log.Printf("Scrape error (%s): %v", url, err)
			continue
		}

		price, err := parse(raw)
		if err != nil {
			log.Printf("Parse error (%s): %v", url, err)
			continue
		}

		fmt.Printf("Successfully scraped: %s\n  Raw: %q\n  Parsed: %.2f €\n\n", url, raw, price)
	}
}
