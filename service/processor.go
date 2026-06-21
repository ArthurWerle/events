package service

import (
	"events/repository"
	"fmt"
	"time"
)

type processorService struct {
	eventRepository repository.EventRepository
}

func NewProcessorService(eventRepository repository.EventRepository) *processorService {
	return &processorService{eventRepository: eventRepository}
}

func (s *processorService) Consume() {
	for {
		fmt.Println("Performing a background task...", time.Now().Format("15:04:05"))
		events, err := s.eventRepository.GetProcessable()

		if err != nil {
			fmt.Printf("Error on lookup: %v", err)
		} else if len(events) == 0 {
			fmt.Println("No events to process...")
		} else {
			fmt.Printf("Found %d event(s) to process\n", len(events))

			for _, event := range events {
				fmt.Printf("Processing event: %d, %s\n", event.ID, event.Payload)
			}
		}

		time.Sleep(5 * time.Second)
	}
}
