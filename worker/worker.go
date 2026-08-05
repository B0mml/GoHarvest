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
	PRICE float64 `json:price`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	connStr := fmt.Sprintf("host=%s port=5432 user=user password=password dbname=itemharvester sslmode=disable", dbHost)

	var db *sql.DB
	var err error

	for range 10 {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Println("Waiting for postgres")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to Database: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

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

	if _, err = db.Exec(createItemsTable); err != nil {
		log.Fatalf("Error creating items table: %v", err)
	}
	if _, err = db.Exec(createHistoryTable); err != nil {
		log.Fatalf("Error creating price_history table: %v", err)
	}

	// Connecting to rabbitmq
	var rabbitCon *amqp.Connection
	for range 10 {
		rabbitCon, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			break
		}
		log.Println("Waiting for RabbitMQ...")
		time.Sleep(2 * time.Second)
	}

	failOnError(err, "Failed to connect to RabbitMQ")
	defer rabbitCon.Close()

	q, err := ch.QueueDeclare("price_items", false, false, false, false, nil)
	failOnError(err, "Error at Queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Error at Consume")

	fmt.Println(" [*] Worker is waitng for msgs...")

	for d := range msgs {
		var article Article
		err := json.Unmarshal(d.Body, &article)
		if err != nil {
			log.Println("Error unpacking json", err)
			continue
		}

		insertSQL := `INSERT INTO articles (title, url) VALUES ($1, $2) ON CONFLICT (url) DO NOTHING;`
		_, err = db.Exec(insertSQL, article.Title, article.URL)
		if err != nil {
			log.Printf("Fehler beim Speichern in Postgres: %v", err)
		} else {
			log.Printf("Artikel in DB gespeichert: %s", article.Title)
		}

	}
}
