package repository

import (
	"context"
	"events/model"
	"log"

	"github.com/jackc/pgx/v5"
)

type EventRepository interface {
	// Create(event *model.Event) error
	// FindByID(id uint) (*model.Event, error)
	FindAll(status *model.Status) ([]model.Event, error)
	// Update(event *model.Event) error
}

type eventRepository struct {
	conn *pgx.Conn
}

func NewEventRepository(db *pgx.Conn) EventRepository {
	return &eventRepository{conn: db}
}

// func (r *eventRepository) Create(event *model.Event) {

// }

// func (r *eventRepository) FindByID(id uint) {

// }

func (r *eventRepository) FindAll(status *model.Status) ([]model.Event, error) {
	query := "SELECT id, payload, status FROM events WHERE status = $1"

	log.Printf("Query: %q", query)
	rows, err := r.conn.Query(context.Background(), query, *status)
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

// func (r *eventRepository) Update(event *model.Event) {

// }
