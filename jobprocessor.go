package events

import (
	"context"
	"events/api/rest"
	"events/db"
	"events/model"
	"events/repository"
	"events/service"
	"events/types"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config holds configuration for the job processor.
type Config struct {
	DatabaseURL     string
	MaxRetries      int
	PollInterval    time.Duration
	EnableDashboard bool
	DashboardPort   int
}

// Processor manages job enqueueing and background processing.
type Processor struct {
	queue           types.Queue
	processor       *service.ProcessorService
	executionRepo   repository.ExecutionRepository
	config          Config
}

// New initialises a Processor: runs DB migrations and opens a connection pool.
func New(cfg Config) (*Processor, error) {
	pool, err := db.InitializeWithURL(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("jobprocessor: db init failed: %w", err)
	}

	eventRepo := repository.NewEventRepository(pool)
	execRepo := repository.NewExecutionRepository(pool)
	queueSvc := service.NewQueueService(eventRepo)
	procSvc := service.NewProcessorService(eventRepo, execRepo, service.ProcessorConfig{
		MaxRetries:   cfg.MaxRetries,
		PollInterval: cfg.PollInterval,
	})

	return &Processor{
		queue:         queueSvc,
		processor:     procSvc,
		executionRepo: execRepo,
		config:        cfg,
	}, nil
}

// Enqueue adds a job to the queue.
func (p *Processor) Enqueue(jobType string, payload string, callbackURL string) (model.Event, error) {
	return p.queue.Enqueue(jobType, payload, callbackURL)
}

// Start launches the background processor goroutine. It also starts the
// dashboard HTTP server when EnableDashboard is true.
func (p *Processor) Start(ctx context.Context) {
	go p.processor.Consume()

	if p.config.EnableDashboard {
		go p.startDashboard()
	}
}

func (p *Processor) startDashboard() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	rest.MountRoutes(r, p.queue, p.executionRepo)
	r.Get("/", rest.ServeUI)

	port := p.config.DashboardPort
	if port == 0 {
		port = 3000
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("jobprocessor dashboard listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Printf("jobprocessor dashboard error: %v", err)
	}
}
