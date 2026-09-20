package api

import (
	"net/http"

	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/services"
)

// Server holds dependencies for HTTP handlers.
type Server struct {
	Store    *db.Store
	Ingestor *services.Ingestor
	Drafter  *services.Drafter
	// DataDir is where settings.yaml and the rest of the editable state live.
	DataDir string
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
