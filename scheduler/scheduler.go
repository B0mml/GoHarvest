package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	dbpkg "github.com/Bommel48/go-scraper-notifier/pkg/db"
	metrics "github.com/Bommel48/go-scraper-notifier/pkg/metrics"
	models "github.com/Bommel48/go-scraper-notifier/pkg/models"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
)

func getOldItems(db *sql.DB) ([]models.Item, error) {
	query := `
		SELECT id, title, url FROM items
		WHERE last_checked_at < NOW() - INTERVAL '5 minutes'
		OR last_checked_at IS NULL;`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ID, &it.Title, &it.URL); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return items, nil
}

func updateLastChecked(db *sql.DB, item models.Item) error {
	_, err := db.Exec(`UPDATE items SET last_checked_at = NOW() WHERE id = $1`, item.ID)
	if err != nil {
		return fmt.Errorf("failed to update last checked item %d: %w", item.ID, err)
	}
	return nil
}

func main() {
	amqpURL := os.Getenv("AMQP_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":2113"
	}

	db, err := dbpkg.Connect(10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

	ch, err := rbmq.Connect(amqpURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer ch.Close()

	go func() {
		err := metrics.StartMetricsServer(metricsAddr)
		if err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		tickStart := time.Now()

		items, err := getOldItems(db)
		if err != nil {
			log.Printf("Error fetching old items: %v", err)
			// FIX 2: Observe tick duration even when query fails
			TickDurationHistogram.Observe(time.Since(tickStart).Seconds())
			continue
		}

		TotalItemsFound.Add(float64(len(items)))

		for _, item := range items {
			scrapeJob := models.ScrapeJob{
				ID:  item.ID,
				URL: item.URL,
			}

			err := rbmq.Publish(ch, rbmq.QueueName, scrapeJob)
			if err != nil {
				TotalPublishedCounter.WithLabelValues("publish_error").Inc()
				log.Printf("Error publishing scrape job %d: %v", scrapeJob.ID, err)
				continue
			}

			if err := updateLastChecked(db, item); err != nil {
				TotalPublishedCounter.WithLabelValues("db_update_error").Inc()
				log.Printf("Error updating last checked for item %d: %v", item.ID, err)
				continue
			}

			log.Printf("%s ready to refresh!\n", item.Title)
			TotalPublishedCounter.WithLabelValues("success").Inc()
		}

		TickDurationHistogram.Observe(time.Since(tickStart).Seconds())
	}
}
