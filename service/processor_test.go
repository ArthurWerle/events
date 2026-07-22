package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ArthurWerle/events/model"

	"github.com/jackc/pgx/v5"
)

// --- in-memory fakes implementing the repository interfaces ---

type fakeEventRepo struct {
	processable []*model.Event
	updated     *model.Event
}

func (f *fakeEventRepo) Create(e *model.Event) (model.Event, error)          { return *e, nil }
func (f *fakeEventRepo) FindByID(id uint) (model.Event, error)               { return model.Event{}, nil }
func (f *fakeEventRepo) FindAll(status *model.Status) ([]model.Event, error) { return nil, nil }

func (f *fakeEventRepo) GetProcessable() (*model.Event, error) {
	if len(f.processable) == 0 {
		return nil, pgx.ErrNoRows
	}
	e := f.processable[0]
	f.processable = f.processable[1:]
	return e, nil
}

func (f *fakeEventRepo) Update(e *model.Event) (model.Event, error) {
	cp := *e
	f.updated = &cp
	return *e, nil
}

type fakeExecRepo struct {
	execs []*model.EventExecution
}

func (f *fakeExecRepo) Create(exec *model.EventExecution) error {
	f.execs = append(f.execs, exec)
	return nil
}

func (f *fakeExecRepo) FindByEventID(eventID uint) ([]model.EventExecution, error) {
	out := []model.EventExecution{}
	for _, e := range f.execs {
		if e.EventID == eventID {
			out = append(out, *e)
		}
	}
	return out, nil
}

func newTestProcessor(er *fakeEventRepo, xr *fakeExecRepo, maxRetries int) *ProcessorService {
	// Near-zero backoff so tests don't actually sleep.
	return NewProcessorService(er, xr, ProcessorConfig{
		MaxRetries:     maxRetries,
		PollInterval:   time.Millisecond,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
}

func statusOf(e *model.Event) model.Status {
	if e == nil {
		return ""
	}
	return e.Status
}

func errOf(x *model.EventExecution) string {
	if x == nil || x.Error == nil {
		return "<nil>"
	}
	return *x.Error
}

// A failing rebuild delivery is requeued as pending (not terminally failed) so
// it will be retried once the callback recovers, and the recorded execution
// error carries the status code + response body for debugging.
func TestProcessNext_RequeuesRebuildJobOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	er := &fakeEventRepo{processable: []*model.Event{
		{ID: 1, JobType: RebuildInsightsJobType, CallbackURL: srv.URL, Payload: "{}"},
	}}
	xr := &fakeExecRepo{}
	p := newTestProcessor(er, xr, 3)

	if err := p.processNext(); err != nil {
		t.Fatalf("processNext: %v", err)
	}

	if got := statusOf(er.updated); got != model.STATUS_PENDING {
		t.Fatalf("want status pending (requeued), got %q", got)
	}
	if len(xr.execs) != 3 {
		t.Fatalf("want 3 recorded executions, got %d", len(xr.execs))
	}
	last := errOf(xr.execs[len(xr.execs)-1])
	if !strings.Contains(last, "500") || !strings.Contains(last, "boom") {
		t.Fatalf("want recorded error to contain status 500 and body %q, got %q", "boom", last)
	}
}

// Non-rebuild job types keep the terminal-failed behavior.
func TestProcessNext_FailsNonRetryableJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	er := &fakeEventRepo{processable: []*model.Event{
		{ID: 2, JobType: "reports-monthly", CallbackURL: srv.URL},
	}}
	xr := &fakeExecRepo{}
	p := newTestProcessor(er, xr, 2)

	if err := p.processNext(); err != nil {
		t.Fatalf("processNext: %v", err)
	}
	if got := statusOf(er.updated); got != model.STATUS_FAILED {
		t.Fatalf("want status failed, got %q", got)
	}
}

// A retryable job that has already exhausted maxDeliveryAttempts is failed
// terminally instead of looping forever.
func TestProcessNext_RebuildFailsAfterMaxDeliveryAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	xr := &fakeExecRepo{}
	for i := 0; i < maxDeliveryAttempts; i++ { // pre-seed prior attempts
		xr.execs = append(xr.execs, &model.EventExecution{EventID: 3})
	}
	er := &fakeEventRepo{processable: []*model.Event{
		{ID: 3, JobType: RebuildInsightsJobType, CallbackURL: srv.URL},
	}}
	p := newTestProcessor(er, xr, 1)

	if err := p.processNext(); err != nil {
		t.Fatalf("processNext: %v", err)
	}
	if got := statusOf(er.updated); got != model.STATUS_FAILED {
		t.Fatalf("want status failed after exceeding maxDeliveryAttempts, got %q", got)
	}
}

// A 2xx response marks the job done on the first attempt.
func TestProcessNext_MarksDoneOnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	er := &fakeEventRepo{processable: []*model.Event{
		{ID: 4, JobType: RebuildInsightsJobType, CallbackURL: srv.URL},
	}}
	xr := &fakeExecRepo{}
	p := newTestProcessor(er, xr, 3)

	if err := p.processNext(); err != nil {
		t.Fatalf("processNext: %v", err)
	}
	if got := statusOf(er.updated); got != model.STATUS_DONE {
		t.Fatalf("want status done, got %q", got)
	}
	if len(xr.execs) != 1 {
		t.Fatalf("want 1 recorded execution, got %d", len(xr.execs))
	}
}
