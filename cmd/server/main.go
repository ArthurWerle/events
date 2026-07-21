package main

import (
	"github.com/ArthurWerle/events/api/rest"
	"github.com/ArthurWerle/events/assets"
	"github.com/ArthurWerle/events/db"
	"github.com/ArthurWerle/events/repository"
	"github.com/ArthurWerle/events/service"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	pool, err := db.InitializeWithURL(dbURL)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	eventRepo := repository.NewEventRepository(pool)
	execRepo := repository.NewExecutionRepository(pool)
	queueSvc := service.NewQueueService(eventRepo)
	procSvc := service.NewProcessorService(eventRepo, execRepo, service.ProcessorConfig{})

	go procSvc.Consume()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	rest.MountRoutes(r, queueSvc, execRepo)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(assets.FS))))
	r.Get("/", rest.ServeUI)

	log.Println("events service listening on :3000")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
