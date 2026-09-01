package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	dbpkg "github.com/Bommel48/go-scraper-notifier/pkg/db"
	"github.com/Bommel48/go-scraper-notifier/pkg/models"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
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

func saveArticle(db *sql.DB, article models.Article) error {
	userID := article.UserID
	if userID == 0 {
		userID = 1
	}

	var itemID int
	insertItemSQL := `
		INSERT INTO items (user_id, title, url) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (user_id, url) DO UPDATE SET title = EXCLUDED.title 
		RETURNING id;`

	err := db.QueryRow(insertItemSQL, userID, article.Title, article.URL).Scan(&itemID)
	if err != nil {
		return fmt.Errorf("failed to upsert item: %w", err)
	}

	insertHistorySQL := `INSERT INTO price_history (item_id, price) VALUES ($1, $2);`
	_, err = db.Exec(insertHistorySQL, itemID, article.Price)
	if err != nil {
		return fmt.Errorf("failed to insert price history: %w", err)
	}

	return nil
}

func processDelivery(db *sql.DB, d amqp.Delivery, id int) {
	var article models.Article
	err := json.Unmarshal(d.Body, &article)
	if err != nil {
		log.Printf("Error unpacking json: %v", err)
		d.Nack(false, false)
		return
	}

	if err := saveArticle(db, article); err != nil {
		log.Printf("Error saving article: %v", err)
		d.Nack(false, true)
		return
	}

	log.Printf("Article & price saved: %s (%.2f €) by worker %d", article.Title, article.Price, id)
	d.Ack(false)
}

func main() {
	var wg sync.WaitGroup

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	connStr := fmt.Sprintf("host=%s port=5432 user=user password=password dbname=itemharvester sslmode=disable", dbHost)
	amqpURL := "amqp://guest:guest@rabbitmq:5672/"

	db, err := connectDB(connStr, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

	if err := dbpkg.InitSchema(db); err != nil {
		log.Fatalf("Schema setup error: %v", err)
	}

	ch, err := rbmq.Connect(amqpURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume(rbmq.QueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error at Consume: %v", err)
	}

	log.Println(" [*] Worker is waiting for msgs...")

	workersStr := os.Getenv("WORKERS")
	numWorkers := 4
	if workersStr != "" && workersStr != "0" {
		n, err := strconv.Atoi(workersStr)
		if err != nil {
			log.Fatalf("Error parsing WORKERS %q: %v", workersStr, err)
		}
		numWorkers = n
	}
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for d := range msgs {
				processDelivery(db, d, id)
			}
		}(i)
	}

	wg.Wait()
}
