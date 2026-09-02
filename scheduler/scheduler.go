package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	dbpkg "github.com/Bommel48/go-scraper-notifier/pkg/db"
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
	amqpURL := "amqp://guest:guest@rabbitmq:5672/"

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

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		items, err := getOldItems(db)
		if err != nil {
			log.Printf("Error fetching old items: %v", err)
			continue
		}

		for _, item := range items {
			scrapeJob := models.ScrapeJob{
				ID:  item.ID,
				URL: item.URL}

			err := rbmq.Publish(ch, rbmq.QueueName, scrapeJob)
			if err != nil {
				log.Printf("Error publishing scrape job %d: %v", scrapeJob.ID, err)
				continue
			}
			log.Printf("%s ready to refresh!\n", item.Title)

			if err := updateLastChecked(db, item); err != nil {
				log.Printf("Error updating last checked for item %d: %v", item.ID, err)
			}
		}
	}
}
