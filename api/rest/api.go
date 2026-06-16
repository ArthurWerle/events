package rest

import (
	"events/model"
	"events/service"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func MountRoutes(r *chi.Mux, queueService service.Queue) (mountedRouter *chi.Mux) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		statusStr := r.URL.Query().Get("status")
		status := model.Status(statusStr)

		if &status != nil && !status.IsValid() {
			http.Error(w, fmt.Sprintf("Status %v not valid.", status), 400)
		}

		events, err := queueService.Lookup(r.Context(), &status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Lookup failed with error %e", err), 404)
		}

		w.Write([]byte(fmt.Sprintf("events:%s", len(events))))
	})

	return r
}
