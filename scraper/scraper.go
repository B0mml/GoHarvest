package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Bommel48/go-scraper-notifier/pkg/models"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func publishArticles(ch *amqp.Channel, queueName string, count int) error {
	log.Println("Starting test...")

	for i := 1; i <= count; i++ {
		article := models.Article{
			UserID: (i % 3) + 1, // Distribute across test user IDs 1, 2, 3
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

func main() {
	amqpURL := "amqp://guest:guest@rabbitmq:5672/"

	ch, err := rbmq.Connect(amqpURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("RabbitMQ connection failure: %v", err)
	}
	defer ch.Close()

	if err := publishArticles(ch, rbmq.QueueName, 5000); err != nil {
		log.Fatalf("Publish error: %v", err)
	}
}
