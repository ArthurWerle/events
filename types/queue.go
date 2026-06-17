package types

import (
	"events/model"
)

type Queue interface {
	Enqueue(event *model.Event) (model.Event, error)
	// Dequeue(ctx context.Context) (model.Event, error)
	Lookup(status *model.Status) ([]model.Event, error)
}
