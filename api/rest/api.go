package rest

import (
	"encoding/json"
	"fmt"
	"github.com/ArthurWerle/events/model"
	"github.com/ArthurWerle/events/repository"
	"github.com/ArthurWerle/events/types"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func MountRoutes(r *chi.Mux, queueService types.Queue, executionRepo repository.ExecutionRepository) *chi.Mux {
	r.Get("/api/events", func(w http.ResponseWriter, r *http.Request) {
		statusStr := r.URL.Query().Get("status")

		if statusStr != "" && !model.Status(statusStr).IsValid() {
			http.Error(w, fmt.Sprintf("Status %v not valid.", statusStr), 400)
			return
		}

		var status model.Status
		if statusStr != "" {
			status = model.Status(statusStr)
		} else {
			status = model.STATUS_PENDING
		}

		events, err := queueService.Lookup(&status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Lookup failed: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	})

	r.Post("/api/events", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			JobType     string `json:"job_type"`
			Payload     string `json:"payload"`
			CallbackURL string `json:"callback_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", 400)
			return
		}

		event, err := queueService.Enqueue(body.JobType, body.Payload, body.CallbackURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Enqueue failed: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(event)
	})

	r.Get("/api/executions", func(w http.ResponseWriter, r *http.Request) {
		eventIDStr := r.URL.Query().Get("event_id")
		if eventIDStr == "" {
			http.Error(w, "event_id query param required", 400)
			return
		}
		eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid event_id", 400)
			return
		}

		execs, err := executionRepo.FindByEventID(uint(eventID))
		if err != nil {
			http.Error(w, fmt.Sprintf("query failed: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execs)
	})

	// Keep backward-compatible /api route pointing to events
	r.Get("/api", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/events?"+r.URL.RawQuery, http.StatusMovedPermanently)
	})

	return r
}
