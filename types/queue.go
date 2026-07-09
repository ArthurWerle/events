package types

import (
	"github.com/ArthurWerle/events/model"
)

type Queue interface {
	Enqueue(jobType string, payload string, callbackURL string) (model.Event, error)
	Lookup(status *model.Status) ([]model.Event, error)
}
