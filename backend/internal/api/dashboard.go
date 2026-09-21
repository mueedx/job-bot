package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/resumes"
)

func (s *Server) handleGetJobDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	detail, err := s.Store.GetJobDetail(id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleUpsertApplication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	var body models.ApplicationUpsert
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var resumePath *string
	if body.Track != nil {
		path, err := resumes.Path(*body.Track)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		resumePath = &path
	}

	app, err := s.Store.UpsertApplication(id, body.CoverLetter, resumePath)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) handleDiscard(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	status := "rejected"
	job, err := s.Store.UpdateJob(id, &models.JobPatch{Status: &status})
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Operator decision: the eligibility engine no longer owns this card.
	_ = s.Store.ClearEligibilityApplied(id)
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	status := "approved"
	job, err := s.Store.UpdateJob(id, &models.JobPatch{Status: &status})
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Operator decision: the eligibility engine no longer owns this card.
	_ = s.Store.ClearEligibilityApplied(id)
	// Honest automation: do not mark applied; submission engine is Phase 5.
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"detail": "Submission dispatcher not implemented yet (Phase 5). Job marked approved for review only — application was NOT sent.",
		"job":    job,
	})
}

func (s *Server) handleSeed(w http.ResponseWriter, _ *http.Request) {
	n, err := s.Store.SeedDemo()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"created": n, "detail": "demo seed complete"})
}

func dailyLimitFromEnv() int {
	raw := os.Getenv("DAILY_APPLICATION_LIMIT")
	if raw == "" {
		return 8
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 8
	}
	return n
}
