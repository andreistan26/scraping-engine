package scraper

import (
	"context"
	"encoding/json"
	"fmt"

	"scraper/internal/log"
	"scraper/internal/types"
	"scraper/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

type WorkerConfig struct {
	Ch     *amqp.Channel
	DbPool *pgxpool.Pool
	//RedisPool *redis.Client
}

func (worker *WorkerConfig) Work(ctx context.Context, workerID int) {
	msgs, err := worker.Ch.Consume(
		"job_queue",
		fmt.Sprintf("%d", workerID),
		false,
		false,
		false,
		false,
		nil,
	)

	utils.Fail(ctx, err, "Failed to consume from queue")

	for msg := range msgs {
		log.FromContext(ctx).Debugf("Worker %d received a message: %s", workerID, msg.Body)

		job := types.Job{}
		json.Unmarshal(msg.Body, &job)

		err = msg.Ack(false)
		if err != nil {
			log.FromContext(ctx).Debugf("Failed to acknowledge message: %v", err)
		}

		err = worker.DoJob(ctx, &job)
		if err != nil {
			log.FromContext(ctx).Debugf("Error occured while running scraper job: %v\n", err)
		}
	}
}
