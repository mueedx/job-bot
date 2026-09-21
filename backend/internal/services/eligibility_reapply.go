package services

import (
	"fmt"

	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
)

// ReapplyResult summarises a rules re-check over stored jobs.
type ReapplyResult struct {
	Checked   int      `json:"checked"`
	Vetoed    int      `json:"vetoed"`
	Restored  int      `json:"restored"`
	Unchanged int      `json:"unchanged"`
	Skipped   int      `json:"skipped"`
	Examples  []string `json:"examples"`
}

// ReapplyEligibility re-runs the rules over stored jobs so a settings change
// takes effect without scraping again.
//
// Ownership rules, so a re-check never overrules the operator:
//   - a job the engine archived is re-archived (or restored when it now passes);
//   - a job the operator moved by hand is counted as skipped and left alone;
//   - a job that is already applied or in interview is never re-archived.
func ReapplyEligibility(store *db.Store, rules *EligibilityRules, limit int) (ReapplyResult, error) {
	var result ReapplyResult
	if store == nil {
		return result, fmt.Errorf("no store configured")
	}
	if limit <= 0 {
		limit = 500
	}
	jobs, err := store.ListJobs("", limit)
	if err != nil {
		return result, err
	}
	policy := rules.WithDefaults()
	result.Examples = []string{}

	for i := range jobs {
		job := &jobs[i]
		verdict := EvaluateEligibility(job, policy)
		owned := job.EligibilityApplied

		applyVerdict := func(applied bool) {
			_ = store.SetJobEligibility(job.ID, db.JobEligibility{
				Status:  verdict.Status,
				Rule:    verdict.Rule,
				Reason:  verdict.Reason,
				Signals: verdict.SignalsJSON(),
				Applied: applied,
			})
		}

		result.Checked++
		switch {
		case verdict.Vetoed():
			switch job.Status {
			case StatusRejected:
				// Already archived (by the engine or by hand) — keep it there.
				applyVerdict(owned)
				result.Unchanged++
			case "applied", "interview":
				// History: never re-archive something you already sent.
				applyVerdict(false)
				result.Skipped++
			default:
				status := StatusRejected
				if _, err := store.UpdateJob(job.ID, &models.JobPatch{Status: &status}); err != nil {
					return result, err
				}
				applyVerdict(true)
				result.Vetoed++
				if len(result.Examples) < 5 {
					result.Examples = append(result.Examples, fmt.Sprintf("%s @ %s — %s", job.Title, job.Company, verdict.Reason))
				}
			}
		case owned:
			// The engine archived this and the rules no longer veto it: restore.
			score := 0.0
			if match, err := store.GetMatch(job.ID); err == nil && match != nil {
				score = match.Score
			}
			status := verdict.StatusAfter(score)
			if _, err := store.UpdateJob(job.ID, &models.JobPatch{Status: &status}); err != nil {
				return result, err
			}
			applyVerdict(false)
			result.Restored++
		default:
			applyVerdict(false)
			result.Unchanged++
		}
	}
	return result, nil
}
