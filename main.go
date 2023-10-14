package main

import (
	"context"
	"os"
	"scraper/internal/api"
	"scraper/internal/scheduler"
	"scraper/internal/scraper"
	"scraper/internal/utils"
	"time"

	"github.com/go-co-op/gocron"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
)

func CreateCommand(ctx context.Context) *cobra.Command {
    cmd := &cobra.Command {
        Use:   "scraper [CMD]",
        Short: "Scraper command line utility.",
        RunE: func(cmd *cobra.Command, args []string) error {
            return nil
        },
    }

    return cmd
}

const DB_MAX_CONN = 32

// Cron jobs, enqueue rabbitmq items
func CreateSheduler(ctx context.Context) *cobra.Command {
    flags := scheduler.SchedulerFlags{}
    cmd := &cobra.Command{
        Use:   "scheduler",
        Short: "runs the scheduler",
        Run: func(cmd *cobra.Command, args []string) {
            dbPool, err := utils.NewDBPool(ctx, DB_MAX_CONN)
            utils.Fail(err, "Unable to connect to database")
            defer dbPool.Close()

            rmqConn, err := amqp.Dial(os.Getenv("RMQ_URL"))
            utils.Fail(err, "Unable to connect to rabbitmq")
            defer rmqConn.Close()


            ch, err := rmqConn.Channel()
            utils.Fail(err, "Unable to make channel")
            defer ch.Close()

            _, err = ch.QueueDeclare(
                "job_queue",
                true,
                false,
                false,
                false,
                nil,
            )
            utils.Fail(err, "Unable to open queue")

            cron := gocron.NewScheduler(time.UTC)
            
            scheduler := scheduler.NewScheduler(
                ctx, 
                scheduler.SchedulerConfig{
                    DbPool:   dbPool,
                    Ch:       ch,
                    Flags:    flags,
                },
            )

            err = scheduler.DbPool.Ping(ctx)
            utils.Fail(err, "Ping error")
            
            cron.Every(1).Seconds().Do(scheduler.Run, ctx)

            cron.StartBlocking()
        },
    }

    cmd.PersistentFlags().BoolVar(&flags.NoResetTimerDB, "noreset", false, "Do not reset scraping timers")

    return cmd
} 

// API server
func CreateAPIServer(ctx context.Context) *cobra.Command {
    cmd := &cobra.Command {
        Use:   "api",
        Short: "runs the api server",
        RunE: func(cmd *cobra.Command, args []string) error {
            dbPool, err := utils.NewDBPool(ctx, DB_MAX_CONN)
            utils.Fail(err, "Unable to connect to database")
            defer dbPool.Close()

            api.StartServer(api.APIServerConfig{
                DbPool: dbPool,
            })

            return nil
        },
    }

    return cmd
}


const (
    WORKER_COUNT = 2
    MAX_CONCURRENT_CONNECTIONS = 10
)

// Scraper worker
func CreateScraper(ctx context.Context) *cobra.Command {
    cmd := &cobra.Command {
        Use:   "scraper",
        Short: "runs the scraper worker",
        Run: func(cmd *cobra.Command, args []string) {
            rmqURL := os.Getenv("RMQ_URL")
            
            conn, err := amqp.Dial(rmqURL)
            utils.Fail(err, "Dial error on worker side")
            defer conn.Close()
            
            ch, err := conn.Channel()
            utils.Fail(err, "Failure when trying to open a RMQ Channel")
            defer ch.Close()

            ch.Qos(MAX_CONCURRENT_CONNECTIONS, 0, false)

            for workerID := 0; workerID < WORKER_COUNT; workerID++ {
                go scraper.NewWorker(ctx, workerID, ch)
            }
            
            select {}
        },
    }

    return cmd
}

func main() {
    ctx := context.Background()
    cmd := CreateCommand(ctx)
    cmd.AddCommand(CreateSheduler(ctx))
    cmd.AddCommand(CreateScraper(ctx))
    cmd.AddCommand(CreateAPIServer(ctx))
    cmd.Execute()
}
