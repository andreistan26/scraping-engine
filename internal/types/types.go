package types

import "time"

type Job struct {
    ID          int32           `json:"job_id"`
    Name        string          `json:"job_name"`
    Frequency   time.Duration   `json:"job_freq"`
    NextScrape  time.Time       `json:"job_next_date"`
    Config      []byte          `json:"job_config"`
}

type Item struct {
    ID          int32           `json:"item_id"`
    Name        string          `json:"item_name"`
    JobID       int32           `json:"item_job_id"`
    Data        []byte          `json:"data"`
}
