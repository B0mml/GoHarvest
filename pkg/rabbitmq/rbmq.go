package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "price_items"

// Connect opens a connection, a channel and declares the queue in one step.
// It returns the channel; the caller closes it (and therefore the connection).
func Connect(amqpURL string, maxRetries int, retryDelay time.Duration) (*amqp.Channel, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(amqpURL)
		if err == nil {
			break
		}
		log.Printf("Waiting for RabbitMQ (attempt %d/%d)...", i+1, maxRetries)
		time.Sleep(retryDelay)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to RabbitMQ after %d attempts: %w", maxRetries, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if _, err := ch.QueueDeclare(QueueName, false, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue %q: %w", QueueName, err)
	}

	return ch, nil
}

// Publish marshals v to JSON and publishes it to the given queue.
func Publish(ch *amqp.Channel, queueName string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
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
		return fmt.Errorf("error publishing message: %w", err)
	}

	return nil
}
