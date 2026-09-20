package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter builds the chi router with CORS and all Phase 1 routes.
func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(180 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", s.handleRoot)
	r.Get("/docs", s.handleDocs)
	r.Get("/openapi.yaml", s.handleOpenAPI)
	r.Get("/health", s.handleHealth)

	r.Route("/api", func(r chi.Router) {
		r.Get("/jobs", s.handleListJobs)
		r.Post("/jobs", s.handleCreateJob)
		r.Get("/jobs/{id}/detail", s.handleGetJobDetail)
		r.Put("/jobs/{id}/application", s.handleUpsertApplication)
		r.Post("/jobs/{id}/discard", s.handleDiscard)
		r.Post("/jobs/{id}/approve", s.handleApprove)
		r.Post("/jobs/{id}/draft", s.handleDraft)
		r.Get("/jobs/{id}", s.handleGetJob)
		r.Patch("/jobs/{id}", s.handleUpdateJob)

		r.Post("/apply/{id}", s.handleApply)
		r.Post("/dev/seed", s.handleSeed)

		r.Post("/ingest/run", s.handleIngestRun)
		r.Get("/ingest/status", s.handleIngestStatus)

		r.Get("/stats", s.handleStats)
		r.Get("/resumes", s.handleListResumes)
		r.Post("/resumes/analyze", s.handleAnalyzeResume)

		r.Get("/sources", s.handleListSources)
		r.Get("/sources/health", s.handleSourcesHealth)

		r.Get("/settings", s.handleGetSettings)
		r.Put("/settings", s.handleSaveSettings)
	})

	return r
}

// allowedOrigins reads CORS_ALLOWED_ORIGINS (comma separated), then falls back
// to DASHBOARD_URL, then to the local dashboard.
func allowedOrigins() []string {
	if raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); raw != "" {
		var origins []string
		for _, part := range strings.Split(raw, ",") {
			if part = strings.TrimSpace(part); part != "" {
				origins = append(origins, part)
			}
		}
		if len(origins) > 0 {
			return origins
		}
	}
	if dashboard := strings.TrimSpace(os.Getenv("DASHBOARD_URL")); dashboard != "" {
		return []string{dashboard}
	}
	return []string{"http://localhost:3000"}
}
