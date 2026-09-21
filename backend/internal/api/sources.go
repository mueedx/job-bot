package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mueedx/job-bot/backend/internal/scrapers"
	"github.com/mueedx/job-bot/backend/internal/services"
)

// loadSettingsFor loads settings for a handler, treating a missing file as
// defaults (the expected first-run state) and anything else as a 500.
func (s *Server) loadSettingsFor(w http.ResponseWriter) (*services.Settings, bool) {
	settings, err := services.LoadSettings(s.DataDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return settings, true
}

// handleGetSettings returns the source toggles, target regions and eligibility
// rules, plus the registry with per-source readiness so the UI can explain why a
// source cannot run (missing key, opt-in flag) and the eligibility defaults the
// settings page offers as suggestions.
func (s *Server) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	settings, ok := s.loadSettingsFor(w)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":             settings,
		"sources":              scrapers.Info(settings.Sources),
		"defaults":             services.DefaultCountries,
		"eligibility_defaults": services.DefaultEligibilityRules(),
		"eligibility_catalog":  services.CountryCatalog(),
	})
}

// handleSaveSettings replaces source toggles, target regions and/or eligibility
// rules. Every field is optional so the UI can save one without the others.
func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	settings, ok := s.loadSettingsFor(w)
	if !ok {
		return
	}
	var body struct {
		Sources            *map[string]bool           `json:"sources"`
		RecruiterCountries []string                   `json:"recruiter_countries"`
		Eligibility        *services.EligibilityRules `json:"eligibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Sources != nil {
		for name := range *body.Sources {
			if _, ok := scrapers.Lookup(name); !ok {
				writeError(w, http.StatusBadRequest, "unknown source "+name)
				return
			}
		}
		settings.Sources = *body.Sources
	}
	if body.RecruiterCountries != nil {
		settings.RecruiterCountries = body.RecruiterCountries
	}
	// Eligibility merges field by field onto the current policy, so the UI can
	// send only what changed and a partial hand-edited file stays valid.
	if body.Eligibility != nil {
		settings.Eligibility = settings.Eligibility.WithDefaults()
		settings.Eligibility.Merge(body.Eligibility)
	}
	if err := services.SaveSettings(s.DataDir, settings); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings": settings,
		"sources":  scrapers.Info(settings.Sources),
	})
}

// handleListSources returns every known source with its metadata and whether it
// is enabled right now (missing API key, opt-in flag not set, or switched off
// in settings). The dashboard uses this to explain why a source is inactive.
func (s *Server) handleListSources(w http.ResponseWriter, _ *http.Request) {
	settings, ok := s.loadSettingsFor(w)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sources": scrapers.Info(settings.Sources),
	})
}

// handleSourcesHealth probes each enabled source and reports what came back, so
// dead board slugs and blocked sites are visible instead of silently empty.
func (s *Server) handleSourcesHealth(w http.ResponseWriter, r *http.Request) {
	if s.Ingestor == nil {
		writeError(w, http.StatusServiceUnavailable, "Job search is not available (ingestor not configured).")
		return
	}
	settings, ok := s.loadSettingsFor(w)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{
		"checked_at": time.Now().UTC(),
		"results":    s.Ingestor.CheckSources(ctx, settings.Sources),
	})
}
