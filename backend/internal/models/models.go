package models

import "time"

// Job is a discovered or tracked job posting.
type Job struct {
	ID           int64      `db:"id" json:"id"`
	Source       string     `db:"source" json:"source"`
	SourceID     string     `db:"source_id" json:"source_id"`
	URL          string     `db:"url" json:"url"`
	Title        string     `db:"title" json:"title"`
	Company      string     `db:"company" json:"company"`
	Location     *string    `db:"location" json:"location"`
	IsRemote     bool       `db:"is_remote" json:"is_remote"`
	IsRelocation bool       `db:"is_relocation" json:"is_relocation"`
	SalaryMin    *int       `db:"salary_min" json:"salary_min"`
	SalaryMax    *int       `db:"salary_max" json:"salary_max"`
	Description  string     `db:"description" json:"description"`
	PostedAt     *time.Time `db:"posted_at" json:"posted_at"`
	Status       string     `db:"status" json:"status"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`

	// Eligibility is the stored verdict of the eligibility rules
	// (pass | veto | unknown) for this posting, with the deciding rule, a
	// human-readable reason and the matched phrases as a JSON array. Nil means
	// the posting was never evaluated (e.g. stored before the rules existed).
	Eligibility        *string `db:"eligibility" json:"eligibility"`
	EligibilityRule    *string `db:"eligibility_rule" json:"eligibility_rule"`
	EligibilityReason  *string `db:"eligibility_reason" json:"eligibility_reason"`
	EligibilitySignals *string `db:"eligibility_signals" json:"eligibility_signals"`
	// EligibilityApplied records that the engine (not the operator) set the
	// current status, so re-checking rules never undoes a manual board move.
	EligibilityApplied bool `db:"eligibility_applied" json:"eligibility_applied"`
}

// Match is the fit score and resume track for a job.
type Match struct {
	ID            int64     `db:"id" json:"id"`
	JobID         int64     `db:"job_id" json:"job_id"`
	Score         float64   `db:"score" json:"score"`
	Track         string    `db:"track" json:"track"`
	MatchedSkills *string   `db:"matched_skills" json:"matched_skills"`
	MissingSkills *string   `db:"missing_skills" json:"missing_skills"`
	ScoreReasons  *string   `db:"score_reasons" json:"score_reasons"`
	ScoredAt      time.Time `db:"scored_at" json:"scored_at"`
}

// Application holds drafted materials and submission state.
type Application struct {
	ID               int64      `db:"id" json:"id"`
	JobID            int64      `db:"job_id" json:"job_id"`
	ResumePath       string     `db:"resume_path" json:"resume_path"`
	CoverLetter      string     `db:"cover_letter" json:"cover_letter"`
	CustomQA         *string    `db:"custom_qa" json:"custom_qa"`
	SubmissionMethod string     `db:"submission_method" json:"submission_method"`
	SubmittedAt      *time.Time `db:"submitted_at" json:"submitted_at"`
	AutoApplied      bool       `db:"auto_applied" json:"auto_applied"`
	SubmissionStatus string     `db:"submission_status" json:"submission_status"`
	ResponseStatus   string     `db:"response_status" json:"response_status"`
	Notes            *string    `db:"notes" json:"notes"`
}

// SubmissionLog records submission pipeline events.
type SubmissionLog struct {
	ID            int64     `db:"id" json:"id"`
	ApplicationID int64     `db:"application_id" json:"application_id"`
	EventType     string    `db:"event_type" json:"event_type"`
	Payload       *string   `db:"payload" json:"payload"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// JobPatch holds optional fields for PATCH /api/jobs/{id}.
type JobPatch struct {
	Status       *string `json:"status"`
	Location     *string `json:"location"`
	IsRemote     *bool   `json:"is_remote"`
	IsRelocation *bool   `json:"is_relocation"`
	SalaryMin    *int    `json:"salary_min"`
	SalaryMax    *int    `json:"salary_max"`
	Title        *string `json:"title"`
	Description  *string `json:"description"`
}

// JobCreate is the body for POST /api/jobs.
type JobCreate struct {
	Source       string  `json:"source"`
	SourceID     string  `json:"source_id"`
	URL          string  `json:"url"`
	Title        string  `json:"title"`
	Company      string  `json:"company"`
	Location     *string `json:"location"`
	IsRemote     *bool   `json:"is_remote"`
	IsRelocation *bool   `json:"is_relocation"`
	SalaryMin    *int    `json:"salary_min"`
	SalaryMax    *int    `json:"salary_max"`
	Description  string  `json:"description"`
	Status       string  `json:"status"`
}

// Stats is the analytics payload for GET /api/stats.
type Stats struct {
	TotalJobs         int            `json:"total_jobs"`
	TotalApplications int            `json:"total_applications"`
	Applied           int            `json:"applied"`
	Interviews        int            `json:"interviews"`
	ByStatus          map[string]int `json:"by_status"`
	DailyLimit        int            `json:"daily_limit"`
}

// JobListItem is a job with optional match fields for board cards.
type JobListItem struct {
	Job
	Score *float64 `db:"score" json:"score"`
	Track *string  `db:"track" json:"track"`
}

// JobDetail bundles job + match + application for the review UI.
type JobDetail struct {
	Job         Job          `json:"job"`
	Match       *Match       `json:"match"`
	Application *Application `json:"application"`
}

// ApplicationUpsert is the body for PUT /api/jobs/{id}/application.
type ApplicationUpsert struct {
	CoverLetter *string `json:"cover_letter"`
	Track       *string `json:"track"`
}

// ResumeProfile is the AI-extracted description of one resume file. Skills and
// Keywords are JSON arrays (same convention as matches.matched_skills).
type ResumeProfile struct {
	ResumePath  string    `db:"resume_path" json:"resume_path"`
	ContentHash string    `db:"content_hash" json:"content_hash"`
	Track       string    `db:"track" json:"track"`
	Skills      *string   `db:"skills" json:"skills"`
	Keywords    *string   `db:"keywords" json:"keywords"`
	Seniority   string    `db:"seniority" json:"seniority"`
	Summary     string    `db:"summary" json:"summary"`
	ExtractedAt time.Time `db:"extracted_at" json:"extracted_at"`
}
