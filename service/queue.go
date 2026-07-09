package service

import (
	"github.com/ArthurWerle/events/model"
	"github.com/ArthurWerle/events/repository"
	"github.com/ArthurWerle/events/types"
)

type queueService struct {
	eventRepository repository.EventRepository
}

func NewQueueService(eventRepository repository.EventRepository) types.Queue {
	return &queueService{eventRepository: eventRepository}
}

func (s *queueService) Lookup(status *model.Status) ([]model.Event, error) {
	return s.eventRepository.FindAll(status)
}

func (s *queueService) Enqueue(jobType string, payload string, callbackURL string) (model.Event, error) {
	event := &model.Event{
		JobType:     jobType,
		Payload:     payload,
		CallbackURL: callbackURL,
	}
	return s.eventRepository.Create(event)
}
