package types

import (
	"events/model"
)

type Queue interface {
	Enqueue(jobType string, payload string, callbackURL string) (model.Event, error)
	Lookup(status *model.Status) ([]model.Event, error)
}
