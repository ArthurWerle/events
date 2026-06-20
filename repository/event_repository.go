package repository

import (
	"context"
	"events/model"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository interface {
	Create(event *model.Event) (model.Event, error)
	FindByID(id uint) (model.Event, error)
	FindAll(status *model.Status) ([]model.Event, error)
	Update(event *model.Event) (model.Event, error)
}

type eventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

func (r *eventRepository) Create(event *model.Event) (model.Event, error) {
	query := "INSERT INTO events (payload, status) VALUES ($1, $2) RETURNING id, payload, status"

	newEvent := model.Event{}
	row := r.pool.QueryRow(context.Background(), query, event.Payload, model.STATUS_PENDING)

	var id uint
	var payload string
	var status model.Status
	err := row.Scan(&id, &payload, &status)

	if err != nil {
		log.Fatal(err)
		return newEvent, err
	}

	newEvent = model.Event{
		ID:      id,
		Payload: payload,
		Status:  status,
	}

	return newEvent, nil
}

func (r *eventRepository) FindByID(id uint) (model.Event, error) {
	query := "SELECT * FROM events WHERE id = $1"

	newEvent := model.Event{}
	rows, err := r.pool.Query(context.Background(), query, id)
	if err != nil {
		log.Fatal(err)
		return newEvent, err
	}

	for rows.Next() {
		var id uint
		var payload string
		var status model.Status
		rows.Scan(&id, &payload, &status)

		newEvent = model.Event{
			ID:      id,
			Payload: payload,
			Status:  status,
		}
	}

	defer rows.Close()

	return newEvent, nil
}

func (r *eventRepository) FindAll(status *model.Status) ([]model.Event, error) {
	query := "SELECT id, payload, status FROM events WHERE status = $1"

	rows, err := r.pool.Query(context.Background(), query, *status)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	events := []model.Event{}

	for rows.Next() {
		var id uint
		var payload string
		var status model.Status
		rows.Scan(&id, &payload, &status)

		event := model.Event{
			ID:      id,
			Payload: payload,
			Status:  status,
		}

		events = append(events, event)
	}

	defer rows.Close()

	return events, nil
}

func (r *eventRepository) Update(event *model.Event) (model.Event, error) {
	query := "UPDATE events SET payload = $1, status = $2 WHERE id = $3"

	newEvent := model.Event{}
	rows, err := r.pool.Query(context.Background(), query, event.Payload, model.STATUS_PENDING, event.ID)
	if err != nil {
		log.Fatal(err)
		return newEvent, err
	}

	for rows.Next() {
		var id uint
		var payload string
		var status model.Status
		rows.Scan(&id, &payload, &status)

		newEvent = model.Event{
			ID:      id,
			Payload: payload,
			Status:  status,
		}
	}

	defer rows.Close()

	return newEvent, nil
}
