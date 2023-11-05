package handler

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
)

// For route '/jobs'
func (h *Handler) JobsHandle(ctx echo.Context) error {
	conn, err := h.DbPool.Acquire(ctx.Request().Context())
	if err != nil {
		ctx.Logger().Errorf("Error when acquiring connection: %v", err)
		return err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx.Request().Context(),
		"SELECT job_id, job_name FROM jobs",
	)
	if err != nil {
		ctx.Logger().Errorf("Error when trying to read jobs: %v", err)
		return err
	}
	defer rows.Close()

	type JobJSON struct {
		ID   int32  `json:"job_id"`
		Name string `json:"job_name"`
	}

	type JobsJSON struct {
		Jobs []JobJSON `json:"jobs"`
	}

	jobs := JobsJSON{}
	for rows.Next() {
		var job JobJSON
		rows.Scan(&job.ID, &job.Name)
		jobs.Jobs = append(jobs.Jobs, job)
	}

	json.NewEncoder(ctx.Response().Writer).Encode(jobs)
	return nil
}

//For router
