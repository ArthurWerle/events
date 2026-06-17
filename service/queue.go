package service

import (
	"events/model"
	"events/repository"
	"events/types"
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

func (s *queueService) Enqueue(event *model.Event) (model.Event, error) {
	return s.eventRepository.Create(event)
}
