package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/services"
)

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}

	if r.URL.Query().Get("enrich") == "1" {
		jobs, err := s.Store.ListJobsEnriched(status, limit, services.JobAgeCutoff())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, jobs)
		return
	}

	jobs, err := s.Store.ListJobs(status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var body models.JobCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Source == "" || body.SourceID == "" || body.URL == "" || body.Title == "" || body.Company == "" || body.Description == "" {
		writeError(w, http.StatusBadRequest, "source, source_id, url, title, company, and description are required")
		return
	}

	isRemote := true
	if body.IsRemote != nil {
		isRemote = *body.IsRemote
	}
	isRelocation := false
	if body.IsRelocation != nil {
		isRelocation = *body.IsRelocation
	}

	job, err := s.Store.CreateJob(&models.Job{
		Source:       body.Source,
		SourceID:     body.SourceID,
		URL:          body.URL,
		Title:        body.Title,
		Company:      body.Company,
		Location:     body.Location,
		IsRemote:     isRemote,
		IsRelocation: isRelocation,
		SalaryMin:    body.SalaryMin,
		SalaryMax:    body.SalaryMax,
		Description:  body.Description,
		Status:       body.Status,
	})
	if errors.Is(err, db.ErrConflict) {
		writeError(w, http.StatusConflict, "Job URL already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var patch models.JobPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	job, err := s.Store.UpdateJob(id, &patch)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}
