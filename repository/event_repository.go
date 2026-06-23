package repository

import (
	"context"
	"events/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository interface {
	Create(event *model.Event) (model.Event, error)
	FindByID(id uint) (model.Event, error)
	FindAll(status *model.Status) ([]model.Event, error)
	GetProcessable() (*model.Event, error)
	Update(event *model.Event) (model.Event, error)
}

type eventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

func (r *eventRepository) Create(event *model.Event) (model.Event, error) {
	query := `INSERT INTO events (payload, status, job_type, callback_url)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id, payload, status, job_type, callback_url`

	var e model.Event
	row := r.pool.QueryRow(context.Background(), query,
		event.Payload, model.STATUS_PENDING, event.JobType, event.CallbackURL)
	err := row.Scan(&e.ID, &e.Payload, &e.Status, &e.JobType, &e.CallbackURL)
	if err != nil {
		return model.Event{}, err
	}
	return e, nil
}

func (r *eventRepository) FindByID(id uint) (model.Event, error) {
	query := `SELECT id, payload, status, job_type, callback_url, created_at FROM events WHERE id = $1`

	var e model.Event
	row := r.pool.QueryRow(context.Background(), query, id)
	err := row.Scan(&e.ID, &e.Payload, &e.Status, &e.JobType, &e.CallbackURL, &e.CreatedAt)
	if err != nil {
		return model.Event{}, err
	}
	return e, nil
}

func (r *eventRepository) FindAll(status *model.Status) ([]model.Event, error) {
	query := `SELECT id, payload, status, job_type, callback_url, created_at FROM events WHERE status = $1`

	rows, err := r.pool.Query(context.Background(), query, *status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.ID, &e.Payload, &e.Status, &e.JobType, &e.CallbackURL, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *eventRepository) GetProcessable() (*model.Event, error) {
	query := `SELECT id, payload, job_type, callback_url
	          FROM events
	          WHERE status = 'pending'
	          ORDER BY id ASC
	          LIMIT 1
	          FOR UPDATE SKIP LOCKED`

	var e model.Event
	row := r.pool.QueryRow(context.Background(), query)
	err := row.Scan(&e.ID, &e.Payload, &e.JobType, &e.CallbackURL)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *eventRepository) Update(event *model.Event) (model.Event, error) {
	query := `UPDATE events SET payload = $1, status = $2 WHERE id = $3`
	_, err := r.pool.Exec(context.Background(), query, event.Payload, event.Status, event.ID)
	if err != nil {
		return model.Event{}, err
	}
	return *event, nil
}
