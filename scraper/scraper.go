package main

import (
	"log"
	"time"

	"github.com/Bommel48/go-scraper-notifier/pkg/models"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func publishPrice(ch *amqp.Channel, title string, url string, price float64) error {
	article := models.Article{
		UserID: 1,
		Title:  title,
		URL:    url,
		Price:  price}

	if err := rbmq.Publish(ch, rbmq.QueueName, article); err != nil {
		log.Printf("Error sending article #%d: %v", article.Title, err)
	}

	time.Sleep(1 * time.Millisecond)

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

}
