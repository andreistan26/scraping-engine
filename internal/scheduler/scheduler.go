package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"scraper/internal/log"
	"scraper/internal/types"
	"scraper/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SchedulerConfig struct {
	DbPool *pgxpool.Pool
	Ch     *amqp.Channel

	Flags SchedulerFlags
}

type SchedulerFlags struct {
	NoResetTimerDB bool
	CreateTables   bool
}

type Scheduler struct {
	SchedulerConfig
	Flags SchedulerFlags
}

func NewScheduler(ctx context.Context, config SchedulerConfig) *Scheduler {
	s := &Scheduler{
		SchedulerConfig: config,
	}

	conn, err := s.DbPool.Acquire(ctx)
	utils.Fail(ctx, err, "Could not aquire db connection")
	defer conn.Release()

	if s.SchedulerConfig.Flags.CreateTables {
		_, err = conn.Exec(ctx, `CREATE TABLE jobs (
            job_id serial PRIMARY KEY,
            job_name VARCHAR(255),
            job_freq INTERVAL,
            job_next_date TIMESTAMP,
            job_config JSON);`,
		)
		utils.Fail(ctx, err, "Failed to make jobs table")

		_, err = conn.Exec(ctx, `CREATE TABLE items (
            item_id serial PRIMARY KEY,
            item_name VARCHAR(255),
            item_data JSON,
            item_job_id INT,
            FOREIGN KEY (item_job_id) REFERENCES jobs(job_id) ON DELETE CASCADE
            );`,
		)
		utils.Fail(ctx, err, "Failed to make items table")
	}

	if !config.Flags.NoResetTimerDB {
		_, err = conn.Exec(ctx, "UPDATE jobs SET job_next_date = now()+job_freq")
		utils.Fail(ctx, err, "Error occured when trying to reset timers")
	}

	return s
}

func (s *Scheduler) Run(ctx context.Context) {
	log.FromContext(ctx).Debugf("Running")

	readConn, err := s.DbPool.Acquire(ctx)
	utils.Fail(ctx, err, "Could not aquire db connection for reading")
	defer readConn.Release()

	writeConn, err := s.DbPool.Acquire(ctx)
	utils.Fail(ctx, err, "Could not aquire db connection for writing")
	defer writeConn.Release()

	rows, err := readConn.Query(ctx, "SELECT job_id, job_name, job_freq, job_config FROM jobs WHERE job_next_date < now()")
	utils.Fail(ctx, err, "Error when trying to get due jobs")

	defer rows.Close()

	for rows.Next() {
		job := types.Job{}

		if err := rows.Scan(&job.ID, &job.Name, &job.Frequency, &job.Config); err != nil {
			fmt.Fprintf(os.Stderr, "Error when scanning row: %v\n", err)
		}

		msg, _ := json.Marshal(job)

		s.Ch.PublishWithContext(
			ctx,
			"",
			"job_queue",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/json",
				Body:        msg,
			},
		)

		tag, err := writeConn.Exec(ctx, "UPDATE jobs SET job_next_date=now()+job_freq WHERE job_id=$1", job.ID)
		if tag.RowsAffected() != 1 {
			fmt.Println("None updated")
		}

		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Printf("Got %v\n", job)
	}
}
