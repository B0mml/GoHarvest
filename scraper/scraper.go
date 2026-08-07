package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Article struct {
	Title string  `json:"title"`
	URL   string  `json:"url"`
	Price float64 `json:"price"`
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

func publishArticles(ch *amqp.Channel, queueName string, count int) error {
	log.Println("Starting test...")

	for i := 1; i <= count; i++ {
		article := Article{
			Title: fmt.Sprintf("Test article #%d", i),
			URL:   fmt.Sprintf("https://example.com/test-%d", i),
			Price: 19.99 + float64(i),
		}

		body, err := json.Marshal(article)
		if err != nil {
			return fmt.Errorf("error marshaling article #%d: %w", i, err)
		}

		err = ch.Publish(
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err != nil {
			log.Printf("Error sending article #%d: %v", i, err)
		}

		time.Sleep(1 * time.Millisecond)
	}

	log.Println("Done!")
	return nil
}

func main() {
	amqpURL := "amqp://guest:guest@rabbitmq:5672/"
	queueName := "price_items"

	conn, err := connectRabbitMQ(amqpURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("RabbitMQ connection failure: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	if err := publishArticles(ch, q.Name, 5000); err != nil {
		log.Fatalf("Publish error: %v", err)
	}
}

