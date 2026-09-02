package main

import (
	"fmt"
	"log"
	"os"
	"time"

	//"github.com/Bommel48/go-scraper-notifier/pkg/models"
	//	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	//amqp "github.com/rabbitmq/amqp091-go"

	models "github.com/Bommel48/go-scraper-notifier/pkg/models"

	_ "github.com/lib/pq"

	"database/sql"
)

func connectDB(connStr string, maxRetries int, retryDelay time.Duration) (*sql.DB, error) {
	var db *sql.DB
	var err error
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				return db, nil
			}
		}
		log.Printf("Waiting for Postgres (attempt %d/%d)...", i+1, maxRetries)
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("could not connect to Postgres after %d attempts: %w", maxRetries, err)
}

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

func main() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	connStr := fmt.Sprintf("host=%s port=5432 user=user password=password dbname=itemharvester sslmode=disable", dbHost)

	db, err := connectDB(connStr, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		items, err := getOldItems(db)
		if err != nil {
			log.Printf("Error fetching old items: %v", err)
			continue
		}

		for _, item := range items {
			log.Printf("%s ready to refresh!\n", item.Title)
		}
	}
}
