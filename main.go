package main

import (
	"events/api/rest"
	"events/db"
	"events/repository"
	"events/service"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	pool := db.Initialize()
	eventRepository := repository.NewEventRepository(pool)
	queueService := service.NewQueueService(eventRepository)

	// Mount API routes
	mountedRoutes := rest.MountRoutes(r, queueService)

	// Mount UI route
	mountedRoutes.Get("/", rest.ServeUI)

	processorService := service.NewProcessorService(eventRepository)
	go processorService.Consume()

	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", mountedRoutes); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
