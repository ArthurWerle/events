package service

import (
	"fmt"
	"github.com/ArthurWerle/events/model"
	"github.com/ArthurWerle/events/repository"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultMaxRetries   = 3
	defaultPollInterval = 5 * time.Second
	defaultHTTPTimeout  = 10 * time.Second
	initialBackoff      = 2 * time.Second
)

type ProcessorConfig struct {
	MaxRetries   int
	PollInterval time.Duration
}

type ProcessorService struct {
	eventRepository     repository.EventRepository
	executionRepository repository.ExecutionRepository
	config              ProcessorConfig
	httpClient          *http.Client
}

func NewProcessorService(
	eventRepository repository.EventRepository,
	executionRepository repository.ExecutionRepository,
	config ProcessorConfig,
) *ProcessorService {
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.PollInterval == 0 {
		config.PollInterval = defaultPollInterval
	}
	return &ProcessorService{
		eventRepository:     eventRepository,
		executionRepository: executionRepository,
		config:              config,
		httpClient:          &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (s *ProcessorService) Consume() {
	for {
		if err := s.processNext(); err != nil {
			log.Printf("processor error: %v", err)
		}
		time.Sleep(s.config.PollInterval)
	}
}

func (s *ProcessorService) processNext() error {
	event, err := s.eventRepository.GetProcessable()
	if err != nil {
		return nil // no rows available
	}

	event.Status = model.STATUS_PROCESSING
	if _, err := s.eventRepository.Update(event); err != nil {
		return fmt.Errorf("failed to mark event %d as processing: %w", event.ID, err)
	}

	log.Printf("processing job id=%d type=%s", event.ID, event.JobType)

	success := false
	backoff := initialBackoff

	for attempt := 1; attempt <= s.config.MaxRetries; attempt++ {
		statusCode, callErr, duration := s.callCallback(event)

		exec := &model.EventExecution{
			EventID:    event.ID,
			DurationMs: &duration,
		}
		if statusCode != 0 {
			exec.StatusCode = &statusCode
		}
		if callErr != "" {
			exec.Error = &callErr
		}

		if logErr := s.executionRepository.Create(exec); logErr != nil {
			log.Printf("failed to record execution for event %d: %v", event.ID, logErr)
		}

		if callErr == "" && statusCode >= 200 && statusCode < 300 {
			success = true
			break
		}

		log.Printf("job id=%d attempt=%d/%d failed (status=%d err=%s), retrying in %s",
			event.ID, attempt, s.config.MaxRetries, statusCode, callErr, backoff)

		if attempt < s.config.MaxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	finalStatus := model.STATUS_DONE
	if !success {
		finalStatus = model.STATUS_FAILED
	}

	event.Status = finalStatus
	if _, err := s.eventRepository.Update(event); err != nil {
		return fmt.Errorf("failed to update event %d to %s: %w", event.ID, finalStatus, err)
	}

	log.Printf("job id=%d finished with status=%s", event.ID, finalStatus)
	return nil
}

func (s *ProcessorService) callCallback(event *model.Event) (int, string, int) {
	params := url.Values{}
	params.Set("job_type", event.JobType)
	params.Set("payload", event.Payload)

	target := event.CallbackURL + "?" + params.Encode()

	start := time.Now()
	resp, err := s.httpClient.Get(target)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		return 0, err.Error(), durationMs
	}
	defer resp.Body.Close()

	return resp.StatusCode, "", durationMs
}
