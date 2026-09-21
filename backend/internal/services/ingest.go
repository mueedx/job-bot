package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/resumes"
	"github.com/mueedx/job-bot/backend/internal/scrapers"
	"gopkg.in/yaml.v3"
)

const ingestLogCap = 100

// IngestLogLine is one progress/log entry for the UI.
type IngestLogLine struct {
	TS      time.Time `json:"ts"`
	Level   string    `json:"level"` // info | warn | error
	Message string    `json:"message"`
}

// IngestStatus is the live / last run summary for the UI.
type IngestStatus struct {
	Running       bool              `json:"running"`
	OK            bool              `json:"ok"`
	Phase         string            `json:"phase"`
	CurrentSource string            `json:"current_source"`
	SourcesTotal  int               `json:"sources_total"`
	SourcesDone   int               `json:"sources_done"`
	LastStarted   *time.Time        `json:"last_started"`
	LastFinished  *time.Time        `json:"last_finished"`
	Inserted      int               `json:"inserted"`
	Scored        int               `json:"scored"`
	Drafted       int               `json:"drafted"`
	SkippedOld    int               `json:"skipped_old"`
	Vetoed        int               `json:"vetoed"`
	SourceErrors  map[string]string `json:"source_errors"`
	Message       string            `json:"message"`
	Logs          []IngestLogLine   `json:"logs"`
}

// Ingestor runs scrapers, persists jobs, matches, and optionally drafts.
type Ingestor struct {
	Store    *db.Store
	DataDir  string
	Client   *http.Client
	Analyzer *ResumeAnalyzer
	Drafter  *Drafter
	Notifier func(job *models.Job, match *models.Match)

	mu     sync.Mutex
	status IngestStatus
}

func (ing *Ingestor) Status() IngestStatus {
	ing.mu.Lock()
	defer ing.mu.Unlock()
	return copyStatus(ing.status)
}

func copyStatus(s IngestStatus) IngestStatus {
	cp := s
	if s.SourceErrors != nil {
		errs := map[string]string{}
		for k, v := range s.SourceErrors {
			errs[k] = v
		}
		cp.SourceErrors = errs
	}
	if s.Logs != nil {
		cp.Logs = append([]IngestLogLine(nil), s.Logs...)
	} else {
		cp.Logs = []IngestLogLine{}
	}
	return cp
}

func (ing *Ingestor) setStatus(mut func(*IngestStatus)) {
	ing.mu.Lock()
	defer ing.mu.Unlock()
	mut(&ing.status)
}

func (ing *Ingestor) logLine(level, msg string) {
	ing.setStatus(func(s *IngestStatus) {
		line := IngestLogLine{TS: time.Now().UTC(), Level: level, Message: msg}
		s.Logs = append(s.Logs, line)
		if len(s.Logs) > ingestLogCap {
			s.Logs = s.Logs[len(s.Logs)-ingestLogCap:]
		}
		s.Message = msg
	})
}

// ErrIngestBusy is returned when StartAsync finds a run already in progress.
var ErrIngestBusy = errors.New("ingest already running")

func (ing *Ingestor) beginRun() (IngestStatus, error) {
	ing.mu.Lock()
	defer ing.mu.Unlock()
	if ing.status.Running {
		st := copyStatus(ing.status)
		st.Message = "ingest already running"
		return st, ErrIngestBusy
	}
	now := time.Now().UTC()
	ing.status = IngestStatus{
		Running:      true,
		OK:           true,
		Phase:        "loading_targets",
		LastStarted:  &now,
		SourceErrors: map[string]string{},
		Logs: []IngestLogLine{{
			TS:      now,
			Level:   "info",
			Message: "Starting job search…",
		}},
		Message: "Starting job search…",
	}
	return copyStatus(ing.status), nil
}

// StartAsync begins a background ingest if idle. Returns current status.
func (ing *Ingestor) StartAsync() (IngestStatus, error) {
	st, err := ing.beginRun()
	if err != nil {
		return st, err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		ing.execute(ctx)
	}()
	return st, nil
}

// Run executes a full ingest + match (+ optional draft) cycle synchronously.
func (ing *Ingestor) Run(ctx context.Context) IngestStatus {
	if _, err := ing.beginRun(); err != nil {
		st := ing.Status()
		st.Message = "ingest already running"
		return st
	}
	return ing.execute(ctx)
}

// execute assumes Running was already claimed via beginRun.
func (ing *Ingestor) execute(ctx context.Context) IngestStatus {
	targets, err := loadTargets(ing.DataDir)
	if err != nil {
		fin := time.Now().UTC()
		ing.setStatus(func(s *IngestStatus) {
			s.Running = false
			s.OK = false
			s.Phase = "error"
			s.LastFinished = &fin
			s.Message = "Could not load company list. Check target_companies.yaml."
		})
		ing.logLine("error", "Could not load company list: "+err.Error())
		return ing.Status()
	}
	ing.logLine("info", "Loaded target company boards")

	client := ing.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	// Sources come from the registry, gated by the user's settings (source
	// toggles and target regions) so adding a source never means editing this
	// loop and a switched-off region really is not fetched.
	settings, err := LoadSettings(ing.DataDir)
	if err != nil {
		ing.logLine("warn", "Could not read settings.yaml — running with defaults ("+err.Error()+").")
		settings = DefaultSettings()
	}

	// Build resume profiles (uses AI extraction when OPENAI_API_KEY is set;
	// falls back to filename/keyword heuristics otherwise). Profiles are
	// cached per content hash, so typically only changed resumes get re-analyzed.
	var profiles []ExtractedProfile
	if ing.Analyzer != nil {
		entries := resumes.Available()
		for _, e := range entries {
			p, perr := ing.Analyzer.AnalyzePDF(ctx, e.Path)
			if perr != nil {
				ing.logLine("warn", fmt.Sprintf("analyze resume %s: %v", e.Path, perr))
				continue
			}
			profiles = append(profiles, *p)
		}
		if len(profiles) > 0 {
			ing.logLine("info", fmt.Sprintf("Prepared %d resume profiles (%s).", len(profiles), profiles[0].Source))
		}
	}

	deps := scrapers.Deps{Client: client, Targets: targets, Countries: settings.RecruiterCountries}
	list, skipped := scrapers.BuildScoped(deps, settings.Sources, settings.RecruiterCountries)
	for _, skip := range skipped {
		ing.logLine("info", skip.Label+" skipped: "+skip.Reason)
	}

	// Eligibility rules apply to every posting in this run; the hint is the
	// operator's own wording for sponsored-relocation roles.
	rules := settings.Eligibility.WithDefaults()
	if !on(rules.Enabled) {
		ing.logLine("info", "Eligibility rules are off — nothing will be filtered.")
	}
	draftOpts := DraftOptions{RelocationHint: rules.RelocationHint}

	ing.setStatus(func(s *IngestStatus) {
		s.Phase = "fetching"
		s.SourcesTotal = len(list)
		s.SourcesDone = 0
	})

	var all []scrapers.RawJob
	for _, sc := range list {
		name := sc.Name()
		ing.setStatus(func(s *IngestStatus) {
			s.CurrentSource = name
			s.Phase = "fetching"
		})
		ing.logLine("info", fmt.Sprintf("Fetching %s…", displaySource(name)))

		srcCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		jobs, err := sc.Fetch(srcCtx)
		cancel()
		if err != nil {
			friendly := friendlySourceError(name, err)
			log.Printf("ingest %s: %v", name, err)
			ing.setStatus(func(s *IngestStatus) {
				s.SourceErrors[name] = err.Error()
				s.SourcesDone++
			})
			ing.logLine("warn", friendly)
			continue
		}
		all = append(all, jobs...)
		ing.setStatus(func(s *IngestStatus) {
			s.SourcesDone++
		})
		ing.logLine("info", fmt.Sprintf("%s: found %d postings", displaySource(name), len(jobs)))
	}

	ing.setStatus(func(s *IngestStatus) {
		s.Phase = "matching"
		s.CurrentSource = ""
	})
	ing.logLine("info", fmt.Sprintf("Matching %d postings…", len(all)))

	inserted, scored, drafted, skippedOld, vetoed, vetoedLogged := 0, 0, 0, 0, 0, 0
	cutoff := JobAgeCutoff()
	if cutoff != nil {
		ing.logLine("info", fmt.Sprintf("Keeping postings from the last %d days only.", JobMaxAgeDays()))
	}
	for i, raw := range all {
		if raw.URL == "" || raw.Title == "" {
			continue
		}
		select {
		case <-ctx.Done():
			fin := time.Now().UTC()
			ing.setStatus(func(s *IngestStatus) {
				s.Running = false
				s.OK = false
				s.Phase = "error"
				s.LastFinished = &fin
				s.Inserted = inserted
				s.Scored = scored
				s.Drafted = drafted
				s.Message = "Search timed out or was cancelled."
			})
			ing.logLine("error", "Search timed out before finishing.")
			return ing.Status()
		default:
		}

		if TooOld(raw.PublishedAt, cutoff) {
			skippedOld++
			continue
		}

		loc := raw.Location
		var locPtr *string
		if loc != "" {
			locPtr = &loc
		}
		job, err := ing.Store.CreateJob(&models.Job{
			Source:      raw.Source,
			SourceID:    raw.SourceID,
			URL:         raw.URL,
			Title:       raw.Title,
			Company:     raw.Company,
			Location:    locPtr,
			IsRemote:    raw.IsRemote,
			SalaryMin:   raw.SalaryMin,
			SalaryMax:   raw.SalaryMax,
			Description: raw.Description,
			PostedAt:    raw.PublishedAt,
			Status:      "discovered",
		})
		if err != nil {
			continue // duplicate or error
		}
		inserted++

		result := MatchJobWithProfiles(job, profiles)
		ms := SkillsJSON(result.MatchedSkills)
		miss := SkillsJSON(result.MissingSkills)
		reasons := result.ScoreReasons
		_ = ing.Store.UpsertMatch(&models.Match{
			JobID:         job.ID,
			Score:         result.Score,
			Track:         result.Track,
			MatchedSkills: &ms,
			MissingSkills: &miss,
			ScoreReasons:  &reasons,
			ScoredAt:      time.Now().UTC(),
		})
		// Eligibility: decide whether this role is worth an application before
		// any effort is spent. A veto archives the posting, skips drafting and
		// skips alerts — that is the whole point of the rules.
		liveness := EvaluateEligibility(job, rules)
		status := result.StatusAfter
		if liveness.Vetoed() {
			status = StatusRejected
			vetoed++
		}
		_ = ing.Store.SetJobEligibility(job.ID, db.JobEligibility{
			Status:  liveness.Status,
			Rule:    liveness.Rule,
			Reason:  liveness.Reason,
			Signals: liveness.SignalsJSON(),
			Applied: liveness.Vetoed(),
		})
		_, _ = ing.Store.UpdateJob(job.ID, &models.JobPatch{Status: &status})
		scored++

		job.Status = status
		job.Eligibility = &liveness.Status
		match, _ := ing.Store.GetMatch(job.ID)

		if liveness.Vetoed() && vetoedLogged < 3 {
			vetoedLogged++
			ing.logLine("info", fmt.Sprintf("Vetoed %s @ %s — %s", job.Title, job.Company, liveness.Reason))
		}

		if !liveness.Vetoed() && ing.Drafter != nil && ing.Drafter.Enabled() && result.Score >= DraftMinScore() {
			ing.setStatus(func(s *IngestStatus) { s.Phase = "drafting" })
			draftOpts.HighlightsRelocation = liveness.HighlightsRelocation
			if err := ing.Drafter.DraftAndSave(ctx, job, match, draftOpts); err != nil {
				log.Printf("draft job %d: %v", job.ID, err)
				ing.logLine("warn", fmt.Sprintf("Draft skipped for %s @ %s", job.Title, job.Company))
			} else {
				drafted++
			}
		}

		if !liveness.Vetoed() && ing.Notifier != nil && match != nil && result.Score >= NotifyMinScore() {
			ing.Notifier(job, match)
		}

		ing.setStatus(func(s *IngestStatus) {
			s.Inserted = inserted
			s.Scored = scored
			s.Drafted = drafted
			s.Vetoed = vetoed
			if s.Phase == "drafting" {
				s.Phase = "matching"
			}
		})

		// Occasional progress log during large batches
		if (i+1)%50 == 0 {
			ing.logLine("info", fmt.Sprintf("Progress: %d new roles scored so far…", scored))
		}
	}

	fin := time.Now().UTC()
	summary := fmt.Sprintf("Found %d new roles (%d scored", inserted, scored)
	if drafted > 0 {
		summary += fmt.Sprintf(", %d drafted", drafted)
	}
	if skippedOld > 0 {
		summary += fmt.Sprintf(", %d skipped as older than %d days", skippedOld, JobMaxAgeDays())
	}
	if vetoed > 0 {
		summary += fmt.Sprintf(", %d vetoed by your eligibility rules — no drafts written", vetoed)
	}
	summary += ")."
	ing.setStatus(func(s *IngestStatus) {
		s.Running = false
		s.OK = true
		s.Phase = "done"
		s.CurrentSource = ""
		s.LastFinished = &fin
		s.Inserted = inserted
		s.Scored = scored
		s.Drafted = drafted
		s.SkippedOld = skippedOld
		s.Vetoed = vetoed
		s.Message = summary
	})
	ing.logLine("info", summary)
	if len(ing.Status().SourceErrors) > 0 {
		ing.logLine("warn", "Some boards had issues — see warnings above. Other sources still completed.")
	}
	return ing.Status()
}

// displaySource is the human label for a source, taken from the source registry
// so the backend and the dashboard can never drift apart.
func displaySource(name string) string { return scrapers.Label(name) }

func friendlySourceError(source string, err error) string {
	msg := err.Error()
	lower := strings.ToLower(msg)
	label := displaySource(source)
	switch {
	case strings.Contains(lower, "404") || strings.Contains(lower, "not found"):
		return fmt.Sprintf("%s: some company boards were not found (check slugs).", label)
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return fmt.Sprintf("%s: timed out — network may be slow.", label)
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host"):
		return fmt.Sprintf("%s: could not reach the board API.", label)
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate"):
		return fmt.Sprintf("%s: rate limited — try again later.", label)
	default:
		return fmt.Sprintf("%s: %s", label, truncateMsg(msg, 120))
	}
}

func truncateMsg(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func loadTargets(dataDir string) (*scrapers.TargetCompanies, error) {
	path := filepath.Join(dataDir, "target_companies.yaml")
	if dataDir == "" {
		path = "data/target_companies.yaml"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read target companies: %w", err)
	}
	var t scrapers.TargetCompanies
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("parse target companies: %w", err)
	}
	return &t, nil
}

// StartIngestCron runs Run on an interval when INGEST_CRON_ENABLED=true.
func StartIngestCron(ing *Ingestor) {
	if os.Getenv("INGEST_CRON_ENABLED") != "true" {
		return
	}
	interval := 24 * time.Hour
	if raw := os.Getenv("INGEST_CRON_INTERVAL_HOURS"); raw != "" {
		if n, err := strconvAtoi(raw); err == nil && n > 0 {
			interval = time.Duration(n) * time.Hour
		}
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			if _, err := ing.StartAsync(); err != nil {
				log.Printf("ingest cron: %v", err)
			}
		}
	}()
	log.Printf("ingest cron enabled every %s", interval)
}

func strconvAtoi(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
