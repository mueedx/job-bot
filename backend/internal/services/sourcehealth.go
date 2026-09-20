package services

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/mueedx/job-bot/backend/internal/scrapers"
)

// SourceHealth is the result of probing one source. Checked is false when the
// source was skipped (not ready, or switched off) — that is reported as skipped
// rather than as a failure, because it is a deliberate state.
type SourceHealth struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Ready      bool   `json:"ready"`
	Reason     string `json:"reason,omitempty"`
	Checked    bool   `json:"checked"`
	OK         bool   `json:"ok"`
	Count      int    `json:"count"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Unverified bool   `json:"unverified"`
}

// healthProbeTimeout bounds a single source so one hanging board cannot stall
// the whole report; sources are probed concurrently.
const healthProbeTimeout = 20 * time.Second

// CheckSources probes every known source and reports what actually came back.
// This is how dead company slugs (boards that no longer exist) become visible
// instead of silently contributing nothing to a search.
func (ing *Ingestor) CheckSources(ctx context.Context, settings map[string]bool) []SourceHealth {
	targets, targetsErr := loadTargets(ing.DataDir)
	client := ing.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	deps := scrapers.Deps{Client: client, Targets: targets}

	specs := scrapers.All()
	results := make([]SourceHealth, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		results[i] = SourceHealth{
			Name:       spec.Name,
			Label:      spec.Label,
			Kind:       string(spec.Kind),
			Unverified: spec.Unverified,
		}
		ready, reason := spec.Ready()
		results[i].Ready = ready
		if !ready {
			results[i].Reason = reason
			continue
		}
		if settings != nil {
			if on, set := settings[spec.Name]; set && !on {
				results[i].Reason = "switched off in settings"
				continue
			}
		}
		// A board source with no slugs makes no requests and returns no jobs
		// without any error, so say plainly what is missing instead of blaming
		// the boards for an empty result.
		if spec.NeedsTargets {
			if targetsErr != nil {
				results[i].Reason = "cannot read target_companies.yaml (" + targetsErr.Error() + ")"
				continue
			}
			if len(spec.Slugs(targets)) == 0 {
				results[i].Reason = "no company slugs configured in target_companies.yaml"
				continue
			}
		}
		wg.Add(1)
		go func(i int, spec scrapers.Spec) {
			defer wg.Done()
			probeCtx, cancel := context.WithTimeout(ctx, healthProbeTimeout)
			defer cancel()
			started := time.Now()
			jobs, err := spec.Build(deps).Fetch(probeCtx)
			results[i].Checked = true
			results[i].Count = len(jobs)
			results[i].DurationMs = time.Since(started).Milliseconds()
			if err != nil {
				results[i].Error = err.Error()
				return
			}
			// A source with no error but no jobs is not healthy — it means the
			// configured slugs are dead, which is the failure mode worth seeing.
			results[i].OK = len(jobs) > 0
			if !results[i].OK {
				results[i].Error = "returned no jobs — check the configured slugs"
			}
		}(i, spec)
	}
	wg.Wait()
	return results
}
