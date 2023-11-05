package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"scraper/internal/api"
	"scraper/internal/cli"
	"scraper/internal/log"
	"scraper/internal/scheduler"
	"scraper/internal/scraper"
	"scraper/internal/utils"

	"github.com/go-co-op/gocron"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
)

func CreateCommand(ctx context.Context) *cobra.Command {
	cliFlags := cli.CliFlags{}
	copts := &cli.CliOptions{}
	cmd := &cobra.Command{
		Use:   "scraper [CMD]",
		Short: "Scraper command line utility.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			for _, opt := range []cli.CliOption{
				cli.WithDefaultLogger(cliFlags.LogLevel),
			} {
				if err := opt(copts); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}

			ctx = log.WithLogger(ctx, copts.Logger)

			cmd.SetContext(ctx)
		},
	}

	cmd.PersistentFlags().StringVar(&cliFlags.LogLevel, "log-level", "debug", "Specify log level {debug,warn,info}")

	return cmd
}

const DB_MAX_CONN = 32

// Cron jobs, enqueue rabbitmq items
func CreateSheduler() *cobra.Command {
	flags := scheduler.SchedulerFlags{}
	cmd := &cobra.Command{
		Use:   "scheduler",
		Short: "runs the scheduler",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			dbPool, err := utils.NewDBPool(ctx, DB_MAX_CONN)
			utils.Fail(ctx, err, "Unable to connect to database")
			defer dbPool.Close()

			rmqConn, err := amqp.Dial(os.Getenv("RMQ_URL"))
			utils.Fail(ctx, err, "Unable to connect to rabbitmq")
			defer rmqConn.Close()

			ch, err := rmqConn.Channel()
			utils.Fail(ctx, err, "Unable to make channel")
			defer ch.Close()

			_, err = ch.QueueDeclare(
				"job_queue",
				true,
				false,
				false,
				false,
				nil,
			)
			utils.Fail(ctx, err, "Unable to open queue")

			cron := gocron.NewScheduler(time.UTC)

			scheduler := scheduler.NewScheduler(
				ctx,
				scheduler.SchedulerConfig{
					DbPool: dbPool,
					Ch:     ch,
					Flags:  flags,
				},
			)

			err = scheduler.DbPool.Ping(ctx)
			utils.Fail(ctx, err, "Ping error")

			log.FromContext(ctx).Debugf("Starting cron jobs")

			cron.Every(1).Seconds().Do(scheduler.Run, ctx)

			cron.StartBlocking()
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.NoResetTimerDB, "noreset", false, "do not reset scraping timers")
	cmd.PersistentFlags().BoolVar(&flags.CreateTables, "create-tables", false, "create tables before running")


	return cmd
}

// API server
func CreateAPIServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "runs the api server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			dbPool, err := utils.NewDBPool(ctx, DB_MAX_CONN)
			utils.Fail(ctx, err, "Unable to connect to database")
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
	WORKER_COUNT               = 2
	MAX_CONCURRENT_CONNECTIONS = 10
	//REDIS_POOL_SIZE = 16
)

// Scraper worker
func CreateScraper() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scraper",
		Short: "runs the scraper worker",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			rmqURL := os.Getenv("RMQ_URL")

			dbPool, err := utils.NewDBPool(ctx, DB_MAX_CONN)
			utils.Fail(ctx, err, "Unable to connect to database")
			defer dbPool.Close()

			conn, err := amqp.Dial(rmqURL)
			utils.Fail(ctx, err, "Dial error on worker side")
			defer conn.Close()

			ch, err := conn.Channel()
			utils.Fail(ctx, err, "Failure when trying to open a RMQ Channel")
			defer ch.Close()

			ch.Qos(MAX_CONCURRENT_CONNECTIONS, 0, false)

			conf := scraper.WorkerConfig{
				Ch:     ch,
				DbPool: dbPool,
			}

			for workerID := 0; workerID < WORKER_COUNT; workerID++ {
				go conf.Work(ctx, workerID)
			}

			select {}
		},
	}

	return cmd
}

func main() {
	ctx := context.Background()
	cmd := CreateCommand(ctx)

	cmd.AddCommand(CreateSheduler())
	cmd.AddCommand(CreateScraper())
	cmd.AddCommand(CreateAPIServer())

	cmd.Execute()
}
