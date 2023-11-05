package utils

import (
	"context"
	"scraper/internal/log"
)

func Fail(ctx context.Context, err error, msg string) {
	if err != nil {
		log.FromContext(ctx).Fatalf("%s: %v\n", msg, err)
	}
}
