package scraper

import (
	"context"
	"os"
	"scraper/internal/types"
	"testing"
)

func TestDoJob(t *testing.T) {
	job := types.Job{}
	file, err := os.Open("test.json")
	if err != nil {
		t.Fatal(err.Error())
	}

	rawConfig := make([]byte, 4096)
	file.Read(rawConfig)
	job.Config = rawConfig
	worker := WorkerConfig{
		nil, nil, nil,
	}

	worker.DoJob(context.Background(), job)
}
