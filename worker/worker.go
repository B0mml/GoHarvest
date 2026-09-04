package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	dbpkg "github.com/Bommel48/go-scraper-notifier/pkg/db"
	metrics "github.com/Bommel48/go-scraper-notifier/pkg/metrics"
	"github.com/Bommel48/go-scraper-notifier/pkg/models"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	"github.com/Bommel48/go-scraper-notifier/pkg/scraper"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func processDelivery(db *sql.DB, d amqp.Delivery, id int) {
	ActiveWorkers.Inc()
	defer ActiveWorkers.Dec()

	var scrapeJob models.ScrapeJob
	err := json.Unmarshal(d.Body, &scrapeJob)
	if err != nil {
		log.Printf("Error unpacking json: %v", err)
		JobsCounter.WithLabelValues("error_json").Inc()
		d.Nack(false, false)
		return
	}

	start := time.Now()
	price, err := scraper.ScrapePrice(scrapeJob.URL)
	ScrapeDuration.Observe(float64(time.Since(start).Seconds()))
	if err != nil {
		log.Printf("Error scraping %s: %v", scrapeJob.URL, err)
		JobsCounter.WithLabelValues("error_scraping").Inc()
		d.Nack(false, false)
		return
	}
	query := `INSERT INTO price_history (item_id, price) VALUES ($1, $2);`
	if _, err := db.Exec(query, scrapeJob.ID, price); err != nil {
		log.Printf("Error saving price: %v", err)
		JobsCounter.WithLabelValues("error_saving").Inc()
		d.Nack(false, true)
		return
	}

	log.Printf("Price saved: %.2f € for item %d by worker %d", price, scrapeJob.ID, id)
	JobsCounter.WithLabelValues("success").Inc()
	d.Ack(false)
}

func main() {
	var wg sync.WaitGroup

	amqpURL := os.Getenv("AMQP_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@rabbitmq:5672/"
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

	msgs, err := ch.Consume(rbmq.QueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error at Consume: %v", err)
	}

	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":2112"
	}
	go func() {
		err := metrics.StartMetricsServer(metricsAddr)
		if err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

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

func StartMetricsServer(metricsAddr string) {
	panic("unimplemented")
}
