package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gocolly/colly/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Article struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")

	if err != nil {
		log.Fatalf("Error connecting to RabbitMQ: %s", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creating channel: %s", err)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hn_articles", // Name der Queue
		false,         // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)

	if err != nil {
		log.Fatalf("Error declaring queue: %s", err)
	}

	fmt.Println("Success connecting to RabbitMQ")

	for {
		c := colly.NewCollector(
			colly.AllowedDomains("news.ycombinator.com"),
		)

		var articles []Article

		c.OnHTML("tr.athing span.titleline > a", func(e *colly.HTMLElement) {
			article := Article{
				Title: e.Text,
				URL:   e.Attr("href"),
			}
			articles = append(articles, article)
		})

		c.OnScraped(func(r *colly.Response) {
			fmt.Printf("Done scraping, %d articles found.\n", len(articles))

			for i, article := range articles {
				if i >= 5 {
					break
				}

				body, err := json.Marshal(article)
				if err != nil {
					log.Println("JSON-Error:", err)
					continue
				}

				err = ch.Publish(
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
					log.Println("Error sending to RabbitMQ:", err)
				} else {
					fmt.Printf("--> Article sent: %s\n", article.Title)
				}
			}
		})

		fmt.Println("Starte Scraping...")
		c.Visit("https://news.ycombinator.com/")
		time.Sleep(30 * time.Second)
	}
}
