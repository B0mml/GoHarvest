package main

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Article struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hn_articles",
		false, false, false, false, nil,
	)
	failOnError(err, "Error at Queue")

	msgs, err := ch.Consume(
		q.Name, // Queue Name
		"",     // Consumer Name
		true,   // Auto-Ack
		false, false, false, nil,
	)
	failOnError(err, "Error at Consume")

	fmt.Println(" [*] Worker is waitng for msgs...")

	for d := range msgs {
		var article Article

		err := json.Unmarshal(d.Body, &article)
		if err != nil {
			log.Println("Error unpacking", err)
			continue
		}

		fmt.Println("------------------------------------")
		fmt.Printf("Received: %s\n", article.Title)
		fmt.Printf("Link: %s\n", article.URL)
	}
}
