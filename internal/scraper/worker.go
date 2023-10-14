package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"scraper/internal/types"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewWorker(ctx context.Context, workerID int, ch *amqp.Channel) {
	msgs, err := ch.Consume(
		"job_queue",                 // Queue name
		fmt.Sprintf("%d", workerID), // Consumer
		false,                       // Auto Acknowledge (set to false for manual acknowledgment)
		false,                       // Exclusive
		false,                       // No Local
		false,                       // No Wait
		nil,                         // Arguments
	)

	if err != nil {
		fmt.Printf("Failed to consume messages: %v", err)
	}

	for msg := range msgs {
		fmt.Printf("Worker %d received a message: %s", workerID, msg.Body)

		job := types.Job{}
		json.Unmarshal(msg.Body, &job)

		DoJob(job)

		// Acknowledge the message to remove it from the queue
		err = msg.Ack(false)
		if err != nil {
			fmt.Printf("Failed to acknowledge message: %v", err)
		}
	}
}
