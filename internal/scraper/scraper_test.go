package scraper

import (
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
    DoJob(job)
}
