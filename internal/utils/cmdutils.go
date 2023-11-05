package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDBPool(ctx context.Context, maxConns int) (*pgxpool.Pool, error) {
	if maxConns < 2 {
		maxConns = 2
	}

	url := fmt.Sprintf(
		"%s?pool_max_conns=%d",
		os.Getenv("DB_URL"),
		maxConns,
	)

	config, err := pgxpool.ParseConfig(url)
	Fail(ctx, err, "Error when parsing DB config")

	config.MaxConnIdleTime = 30 * time.Second

	return pgxpool.NewWithConfig(ctx, config)
}
