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
	PRICE float64 `json:price`
}

func main() {
	var conn *amqp.Connection
	var err error

	for range 10 {
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			break
		}
		log.Println("Warte auf RabbitMQ...")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Konnte nicht mit RabbitMQ verbinden: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("hn_articles", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Start Test")

	for i := 1; i <= 5000; i++ {
		article := Article{
			Title: fmt.Sprintf("Test Artikel #%d", i),
			URL:   fmt.Sprintf("https://example.com/test-%d", i),
		}

		body, _ := json.Marshal(article)

		err := ch.Publish(
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err != nil {
			log.Printf("Fehler beim Senden: %v", err)
		}

		// short sleep for testing
		time.Sleep(1 * time.Millisecond)
	}

	log.Println("Done!")
}
