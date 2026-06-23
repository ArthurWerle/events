package main

import (
	"context"
	"log"
	"os"
	"time"

	events "events"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	p, err := events.New(events.Config{
		DatabaseURL:     dbURL,
		MaxRetries:      3,
		PollInterval:    5 * time.Second,
		EnableDashboard: true,
		DashboardPort:   3000,
	})
	if err != nil {
		log.Fatalf("failed to create job processor: %v", err)
	}

	ctx := context.Background()
	p.Start(ctx)

	// Example: enqueue a job at startup (remove in production)
	event, err := p.Enqueue("example_job", `{"hello":"world"}`, "http://localhost:9000/callback")
	if err != nil {
		log.Printf("enqueue error: %v", err)
	} else {
		log.Printf("enqueued job id=%d", event.ID)
	}

	// Block forever
	select {}
}
