package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"scraper/internal/types"
	"scraper/internal/utils"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/text/message"
)

type SchedulerConfig struct {
    DbPool   *pgxpool.Pool
    Ch       *amqp.Channel
    
    Flags   SchedulerFlags
}

type SchedulerFlags struct {
    NoResetTimerDB bool     
}

type Scheduler struct {
    SchedulerConfig
    Flags  SchedulerFlags
}

func NewScheduler(ctx context.Context, config SchedulerConfig) *Scheduler {
    s := &Scheduler{
        SchedulerConfig: config,
    }


    if !config.Flags.NoResetTimerDB {
        fmt.Println("resseting")
        
        conn, err := s.DbPool.Acquire(ctx)
        utils.Fail(err, "Could not aquire db connection")
        defer conn.Release()
     
        _, err = conn.Exec(ctx, "UPDATE jobs SET job_next_date = now()+job_freq")
        utils.Fail(err, "Error occured when trying to reset timers")
    }

    return s
}

func (s *Scheduler) Run(ctx context.Context) {
    fmt.Println("running")
    readConn, err := s.DbPool.Acquire(ctx)
    utils.Fail(err, "Could not aquire db connection for reading")
    defer readConn.Release()
    
    writeConn, err := s.DbPool.Acquire(ctx)
    utils.Fail(err, "Could not aquire db connection for writing")
    defer writeConn.Release()

    rows, err := readConn.Query(ctx, "SELECT job_id, job_name, job_freq, job_config FROM jobs WHERE job_next_date < now()")
    utils.Fail(err, "Error when trying to get due jobs")

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
                Body: msg,
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
