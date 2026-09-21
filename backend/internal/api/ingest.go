package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/services"
)

func (s *Server) handleIngestRun(w http.ResponseWriter, r *http.Request) {
	if s.Ingestor == nil {
		writeError(w, http.StatusServiceUnavailable, "Job search is not available (ingestor not configured).")
		return
	}
	st, err := s.Ingestor.StartAsync()
	if errors.Is(err, services.ErrIngestBusy) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"detail": "A search is already running.",
			"status": st,
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, st)
}

func (s *Server) handleIngestStatus(w http.ResponseWriter, _ *http.Request) {
	if s.Ingestor == nil {
		writeError(w, http.StatusServiceUnavailable, "Job search is not available (ingestor not configured).")
		return
	}
	writeJSON(w, http.StatusOK, s.Ingestor.Status())
}

func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	if s.Drafter == nil || !s.Drafter.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "OPENAI_API_KEY not configured")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	job, err := s.Store.GetJob(id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	match, _ := s.Store.GetMatch(id)
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	opts := services.DraftOptions{}
	if settings, err := services.LoadSettings(s.DataDir); err == nil {
		rules := settings.Eligibility.WithDefaults()
		opts.RelocationHint = rules.RelocationHint
		verdict := services.EvaluateEligibility(job, rules)
		opts.HighlightsRelocation = verdict.HighlightsRelocation
	}
	if err := s.Drafter.DraftAndSave(ctx, job, match, opts); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	detail, err := s.Store.GetJobDetail(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
