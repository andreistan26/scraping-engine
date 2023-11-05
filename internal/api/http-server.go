package api

import (
	"scraper/internal/api/handler"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type APIServerConfig struct {
	DbPool *pgxpool.Pool
}

func StartServer(config APIServerConfig) {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	h := &handler.Handler{
		DbPool: config.DbPool,
	}

	e.GET("/jobs", h.JobsHandle)

	e.Start(":6942")
}
