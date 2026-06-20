package main

import (
	"events/api/rest"
	"events/db"
	"events/repository"
	"events/service"
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

	mountedRoutes := rest.MountRoutes(r, queueService)
	http.ListenAndServe(":3000", mountedRoutes)
}
