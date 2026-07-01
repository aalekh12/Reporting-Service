package handler

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"reporting-service/internal/middleware"
)

// NewRouter wires all routes and middleware for the service.
func NewRouter(svc ReportServicer, log *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.Recover(log))
	r.Use(middleware.Logging(log))

	r.Get("/healthz", Health)

	h := NewReportHandler(svc)
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Get("/", h.ListReports)
		r.Get("/{id}", h.GetReport)
		r.Post("/{id}/preview", h.Preview)
		r.Post("/{id}/generate", h.Generate)
		r.Post("/{id}/export", h.Export)
	})

	return r
}
