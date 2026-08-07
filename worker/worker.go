package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Article struct {
	Title string  `json:"title"`
	URL   string  `json:"url"`
	Price float64 `json:"price"`
}

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

func initSchema(db *sql.DB) error {
	createItemsTable := `
	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		url TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	createHistoryTable := `
	CREATE TABLE IF NOT EXISTS price_history (
		id SERIAL PRIMARY KEY,
		item_id INT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		price NUMERIC(10, 2) NOT NULL,
		recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createItemsTable); err != nil {
		return fmt.Errorf("error creating items table: %w", err)
	}
	if _, err := db.Exec(createHistoryTable); err != nil {
		return fmt.Errorf("error creating price_history table: %w", err)
	}
	return nil
}

func connectRabbitMQ(amqpURL string, maxRetries int, retryDelay time.Duration) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(amqpURL)
		if err == nil {
			return conn, nil
		}
		log.Printf("Waiting for RabbitMQ (attempt %d/%d)...", i+1, maxRetries)
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("could not connect to RabbitMQ after %d attempts: %w", maxRetries, err)
}

func saveArticle(db *sql.DB, article Article) error {
	var itemID int
	insertItemSQL := `
		INSERT INTO items (title, url) 
		VALUES ($1, $2) 
		ON CONFLICT (url) DO UPDATE SET title = EXCLUDED.title 
		RETURNING id;`

	err := db.QueryRow(insertItemSQL, article.Title, article.URL).Scan(&itemID)
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

func processDelivery(db *sql.DB, d amqp.Delivery) {
	var article Article
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

	log.Printf("Article & price saved: %s (%.2f €)", article.Title, article.Price)
	d.Ack(false)
}

func main() {
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

	if err := initSchema(db); err != nil {
		log.Fatalf("Schema setup error: %v", err)
	}

	rabbitCon, err := connectRabbitMQ(amqpURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer rabbitCon.Close()

	ch, err := rabbitCon.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("price_items", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error declaring Queue: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error at Consume: %v", err)
	}

	log.Println(" [*] Worker is waiting for msgs...")

	for d := range msgs {
		processDelivery(db, d)
	}
}

