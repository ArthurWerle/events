package service

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ArthurWerle/events/model"
	"github.com/ArthurWerle/events/repository"

	"github.com/jackc/pgx/v5"
)

const (
	// defaultMaxRetries is the number of delivery attempts per processing pass.
	// With the default backoff a pass spans ~60s (2+4+8+16+30), enough to ride
	// out a callback target that is briefly restarting (e.g. one that runs DB
	// migrations before it starts listening).
	defaultMaxRetries   = 6
	defaultPollInterval = 5 * time.Second
	// Callbacks run the job synchronously (e.g. reports generates and emails
	// the report inside the request), so allow well beyond the reports-side
	// 90s pipeline timeout.
	defaultHTTPTimeout    = 120 * time.Second
	defaultInitialBackoff = 2 * time.Second
	defaultMaxBackoff     = 30 * time.Second

	// RebuildInsightsJobType mirrors the transactions job type of the same
	// name. Delivery of this job is kept retryable across poll cycles (requeued
	// as pending on exhaustion) so a transient outage of the insights callback
	// doesn't permanently drop a spending-insight rebuild.
	RebuildInsightsJobType = "rebuild-spending-insights"

	// maxDeliveryAttempts bounds the total delivery attempts (across poll
	// cycles) for a retryable job before it is marked failed, so a
	// permanently-unreachable callback can't be retried forever.
	maxDeliveryAttempts = 60

	// responseBodySnippetBytes caps how much of a non-2xx response body is
	// captured into the execution error for debugging.
	responseBodySnippetBytes = 512
)

type ProcessorConfig struct {
	MaxRetries     int
	PollInterval   time.Duration
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
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
	if config.InitialBackoff == 0 {
		config.InitialBackoff = defaultInitialBackoff
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = defaultMaxBackoff
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // no rows available
		}
		return fmt.Errorf("failed to fetch processable event: %w", err)
	}

	event.Status = model.STATUS_PROCESSING
	if _, err := s.eventRepository.Update(event); err != nil {
		return fmt.Errorf("failed to mark event %d as processing: %w", event.ID, err)
	}

	log.Printf("processing job id=%d type=%s", event.ID, event.JobType)

	success := false
	lastErr := ""
	backoff := s.config.InitialBackoff

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
			lastErr = callErr
		}

		if logErr := s.executionRepository.Create(exec); logErr != nil {
			log.Printf("failed to record execution for event %d: %v", event.ID, logErr)
		}

		if callErr == "" && statusCode >= 200 && statusCode < 300 {
			success = true
			break
		}

		if attempt < s.config.MaxRetries {
			log.Printf("job id=%d type=%s attempt=%d/%d failed (status=%d err=%q), retrying in %s",
				event.ID, event.JobType, attempt, s.config.MaxRetries, statusCode, callErr, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > s.config.MaxBackoff {
				backoff = s.config.MaxBackoff
			}
		}
	}

	// A failed delivery of a retryable job is requeued as pending (retried on a
	// later poll cycle) rather than terminally failed, so a transient callback
	// outage doesn't permanently drop the job. maxDeliveryAttempts bounds the
	// total attempts so a permanently-unreachable callback still gives up.
	finalStatus := model.STATUS_DONE
	attempts := 0
	if !success {
		attempts = s.attemptCount(event)
		if s.isRetryable(event) && attempts < maxDeliveryAttempts {
			finalStatus = model.STATUS_PENDING
		} else {
			finalStatus = model.STATUS_FAILED
		}
	}

	event.Status = finalStatus
	if _, err := s.eventRepository.Update(event); err != nil {
		return fmt.Errorf("failed to update event %d to %s: %w", event.ID, finalStatus, err)
	}

	switch finalStatus {
	case model.STATUS_DONE:
		log.Printf("job id=%d type=%s finished with status=done", event.ID, event.JobType)
	case model.STATUS_PENDING:
		log.Printf("job id=%d type=%s delivery failing — requeued as pending for later retry (attempts so far=%d, last error: %s)",
			event.ID, event.JobType, attempts, lastErr)
	default:
		log.Printf("job id=%d type=%s delivery failed after %d attempts (last error: %s)",
			event.ID, event.JobType, attempts, lastErr)
	}
	return nil
}

// isRetryable reports whether a failed delivery of this job should be retried
// on a later poll cycle (status set back to pending) instead of terminally
// failed. Only the insight-rebuild job is retryable; other job types keep the
// terminal-failed behavior.
func (s *ProcessorService) isRetryable(event *model.Event) bool {
	return event.JobType == RebuildInsightsJobType
}

// attemptCount returns how many delivery attempts have been recorded for the
// event so far (from event_executions), used to bound cross-cycle retries
// without a schema change. On a lookup error it returns maxDeliveryAttempts so
// the caller fails closed (no further requeue) rather than risking an endless
// loop.
func (s *ProcessorService) attemptCount(event *model.Event) int {
	execs, err := s.executionRepository.FindByEventID(event.ID)
	if err != nil {
		log.Printf("job id=%d: could not count executions, treating as exhausted: %v", event.ID, err)
		return maxDeliveryAttempts
	}
	return len(execs)
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
		// Transport-level failure (DNS, connection refused, timeout, ...). Go
		// already embeds the URL in err; add the job type so the recorded
		// execution row is self-describing.
		return 0, fmt.Sprintf("job_type=%s: %v", event.JobType, err), durationMs
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Capture a bounded snippet of the body so the reason for a non-2xx
		// (e.g. an error message from the callback) is recorded instead of nil.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, responseBodySnippetBytes))
		msg := fmt.Sprintf("callback %s returned %d for job_type=%s",
			event.CallbackURL, resp.StatusCode, event.JobType)
		if body := strings.TrimSpace(string(snippet)); body != "" {
			msg += ": " + body
		}
		return resp.StatusCode, msg, durationMs
	}

	return resp.StatusCode, "", durationMs
}
