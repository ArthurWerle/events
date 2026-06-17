package rest

import (
	"encoding/json"
	"events/model"
	"events/types"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func MountRoutes(r *chi.Mux, queueService types.Queue) (mountedRouter *chi.Mux) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, fmt.Sprintf("Lookup failed with error %e", err), 404)
		}

		eventsJson, err := json.Marshal(events)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing events to JSON: %e", err), 500)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(eventsJson)
	})

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var event model.Event

		json.NewDecoder(r.Body).Decode(&event)

		event, err := queueService.Enqueue(&event)
		if err != nil {
			http.Error(w, fmt.Sprintf("Lookup failed with error %e", err), 404)
		}

		eventJson, err := json.Marshal(event)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing event to JSON: %e", err), 500)
		}

		w.Header().Set("Content-Type", "application/json")

		w.Write(eventJson)
	})

	return r
}
