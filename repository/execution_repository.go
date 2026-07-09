package repository

import (
	"context"
	"github.com/ArthurWerle/events/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExecutionRepository interface {
	Create(exec *model.EventExecution) error
	FindByEventID(eventID uint) ([]model.EventExecution, error)
}

type executionRepository struct {
	pool *pgxpool.Pool
}

func NewExecutionRepository(pool *pgxpool.Pool) ExecutionRepository {
	return &executionRepository{pool: pool}
}

func (r *executionRepository) Create(exec *model.EventExecution) error {
	query := `INSERT INTO event_executions (event_id, status_code, error, duration_ms)
	          VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(context.Background(), query,
		exec.EventID, exec.StatusCode, exec.Error, exec.DurationMs)
	return err
}

func (r *executionRepository) FindByEventID(eventID uint) ([]model.EventExecution, error) {
	query := `SELECT id, event_id, attempted_at, status_code, error, duration_ms
	          FROM event_executions WHERE event_id = $1 ORDER BY attempted_at ASC`

	rows, err := r.pool.Query(context.Background(), query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []model.EventExecution
	for rows.Next() {
		var e model.EventExecution
		if err := rows.Scan(&e.ID, &e.EventID, &e.AttemptedAt, &e.StatusCode, &e.Error, &e.DurationMs); err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, nil
}
