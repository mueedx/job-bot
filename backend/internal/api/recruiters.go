package api

import (
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/recruiters"
	"github.com/mueedx/job-bot/backend/internal/services"
)

// handleListRecruiters returns the recruiter directory filtered to the user's
// enabled countries and optional track filter. It merges the shipped catalog
// with data/recruiters.yaml (user overrides/disables/adds) at load time.
//
// Query params:
//   ?track=<comma-separated track tokens, e.g. fullstack,blockchain>
//   ?countries=<comma-separated country codes, e.g. ie,gb,pk>
//
// When both are absent the full filtered-by-settings list is returned.
func (s *Server) handleListRecruiters(w http.ResponseWriter, r *http.Request) {
	settings, err := services.LoadSettings(s.DataDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot read settings: "+err.Error())
		return
	}
	rs := recruiters.Load(s.DataDir)
	// Default to enabled countries from settings, not the full catalog.
	enabled := settings.RecruiterCountries
	if q := r.URL.Query().Get("countries"); q != "" {
		enabled = strings.Split(strings.TrimSpace(q), ",")
		for i := range enabled {
			enabled[i] = strings.TrimSpace(enabled[i])
		}
	}
	rs = recruiters.FilterByCountries(rs, enabled)

	track := r.URL.Query().Get("track")
	if track != "" {
		tokens := strings.Split(strings.TrimSpace(track), ",")
		for i := range tokens {
			tokens[i] = strings.TrimSpace(tokens[i])
		}
		rs = recruiters.FilterByField(rs, tokens)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"recruiters":          rs,
		"grouped_by_country":  recruiters.GroupByCountry(rs),
		"shipped_count":       len(recruiters.Catalog),
		"countries_filtered_to": enabled,
	})
}
