package service

import (
	"context"
	"events/model"
	"events/repository"
)

type Queue interface {
	// Enqueue(ctx context.Context, payload []byte) error
	// Dequeue(ctx context.Context) error
	// Ack(ctx context.Context, id string) error
	Lookup(ctx context.Context, status *model.Status) ([]model.Event, error)
}

type queueService struct {
	eventRepository repository.EventRepository
}

func NewQueueService(eventRepository repository.EventRepository) Queue {
	return &queueService{eventRepository: eventRepository}
}

func (s *queueService) Lookup(ctx context.Context, status *model.Status) ([]model.Event, error) {
	return s.eventRepository.FindAll(status)
}
