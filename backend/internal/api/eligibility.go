package api

import (
	"encoding/json"
	"net/http"

	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/services"
)

// eligibilityBody is the optional payload for the preview and re-check
// endpoints. Supplying `rules` evaluates against unsaved edits, so the settings
// page can test a policy before saving it.
type eligibilityBody struct {
	Title       string                     `json:"title"`
	Location    string                     `json:"location"`
	Description string                     `json:"description"`
	IsRemote    *bool                      `json:"is_remote"`
	Rules       *services.EligibilityRules `json:"rules"`
}

// rulesFor merges optional submitted rules onto the saved policy.
func rulesFor(saved *services.EligibilityRules, over *services.EligibilityRules) *services.EligibilityRules {
	policy := saved.WithDefaults()
	if over != nil {
		policy.Merge(over)
		policy.Normalize()
	}
	return policy
}

// handlePreviewEligibility evaluates one posting against the eligibility rules
// and returns the verdict. Nothing is stored — this backs the tester on the
// settings page (and the same code path the ingest loop uses).
func (s *Server) handlePreviewEligibility(w http.ResponseWriter, r *http.Request) {
	var body eligibilityBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Title == "" && body.Description == "" && body.Location == "" {
		writeError(w, http.StatusBadRequest, "provide a title, location or description to test")
		return
	}
	settings, ok := s.loadSettingsFor(w)
	if !ok {
		return
	}
	job := &models.Job{Title: body.Title, Description: body.Description, IsRemote: true}
	if body.Location != "" {
		loc := body.Location
		job.Location = &loc
	}
	if body.IsRemote != nil {
		job.IsRemote = *body.IsRemote
	}
	policy := rulesFor(settings.Eligibility, body.Rules)
	writeJSON(w, http.StatusOK, map[string]any{
		"verdict": services.EvaluateEligibility(job, policy),
		"rules":   policy,
	})
}

// handleReapplyEligibility re-runs the rules over stored jobs, so a policy change
// takes effect without scraping again. Cards the operator moved by hand are left
// alone (see services.ReapplyEligibility).
func (s *Server) handleReapplyEligibility(w http.ResponseWriter, r *http.Request) {
	var body eligibilityBody
	_ = json.NewDecoder(r.Body).Decode(&body) // body is optional

	settings, ok := s.loadSettingsFor(w)
	if !ok {
		return
	}
	policy := rulesFor(settings.Eligibility, body.Rules)
	result, err := services.ReapplyEligibility(s.Store, policy, 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}
