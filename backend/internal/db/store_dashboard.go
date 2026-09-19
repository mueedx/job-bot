package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/resumes"
)

// GetMatch returns the match for a job, or ErrNotFound.
func (s *Store) GetMatch(jobID int64) (*models.Match, error) {
	var m models.Match
	err := s.db.Get(&m, `
		SELECT id, job_id, score, track, matched_skills, missing_skills, score_reasons, scored_at
		FROM matches WHERE job_id = ?`, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get match: %w", err)
	}
	return &m, nil
}

// GetApplication returns the application for a job, or ErrNotFound.
func (s *Store) GetApplication(jobID int64) (*models.Application, error) {
	var a models.Application
	err := s.db.Get(&a, `
		SELECT id, job_id, resume_path, cover_letter, custom_qa, submission_method,
		       submitted_at, auto_applied, submission_status, response_status, notes
		FROM applications WHERE job_id = ?`, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	return &a, nil
}

// GetJobDetail returns job with optional match and application.
func (s *Store) GetJobDetail(jobID int64) (*models.JobDetail, error) {
	job, err := s.GetJob(jobID)
	if err != nil {
		return nil, err
	}
	detail := &models.JobDetail{Job: *job}
	if m, err := s.GetMatch(jobID); err == nil {
		detail.Match = m
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if a, err := s.GetApplication(jobID); err == nil {
		detail.Application = a
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return detail, nil
}

// ListJobsEnriched returns jobs with optional match score/track.
// postedAfter, when non-nil, hides postings older than that instant
// (postings with an unknown date are always kept).
func (s *Store) ListJobsEnriched(status string, limit int, postedAfter *time.Time) ([]models.JobListItem, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	query := `
		SELECT j.id, j.source, j.source_id, j.url, j.title, j.company, j.location,
		       j.is_remote, j.is_relocation, j.salary_min, j.salary_max,
		       j.description, j.posted_at, j.status, j.created_at,
		       m.score AS score, m.track AS track
		FROM jobs j
		LEFT JOIN matches m ON m.job_id = j.id`
	conditions := []string{}
	args := []any{}
	if status != "" {
		conditions = append(conditions, "j.status = ?")
		args = append(args, status)
	}
	// Job age window: jobs without a known date are always kept.
	if postedAfter != nil {
		conditions = append(conditions, "(j.posted_at IS NULL OR j.posted_at >= ?)")
		args = append(args, postedAfter.UTC())
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY j.created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs enriched: %w", err)
	}
	defer rows.Close()

	items := []models.JobListItem{}
	for rows.Next() {
		var row struct {
			models.Job
			Score sql.NullFloat64 `db:"score"`
			Track sql.NullString  `db:"track"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan enriched job: %w", err)
		}
		item := models.JobListItem{Job: row.Job}
		if row.Score.Valid {
			v := row.Score.Float64
			item.Score = &v
		}
		if row.Track.Valid {
			v := row.Track.String
			item.Track = &v
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpsertApplication creates or updates an application draft for a job.
func (s *Store) UpsertApplication(jobID int64, coverLetter *string, resumePath *string) (*models.Application, error) {
	if _, err := s.GetJob(jobID); err != nil {
		return nil, err
	}

	existing, err := s.GetApplication(jobID)
	if errors.Is(err, ErrNotFound) {
		letter := ""
		if coverLetter != nil {
			letter = *coverLetter
		}
		path, pathErr := resumes.Path("fullstack")
		if pathErr != nil {
			path = "resumes/fullstack.pdf"
		}
		if resumePath != nil && *resumePath != "" {
			path = *resumePath
		}
		_, err = s.db.Exec(`
			INSERT INTO applications (
				job_id, resume_path, cover_letter, submission_method,
				auto_applied, submission_status, response_status
			) VALUES (?, ?, ?, 'manual', 0, 'pending', 'no_reply')`,
			jobID, path, letter,
		)
		if err != nil {
			return nil, fmt.Errorf("insert application: %w", err)
		}
		return s.GetApplication(jobID)
	}
	if err != nil {
		return nil, err
	}

	letter := existing.CoverLetter
	if coverLetter != nil {
		letter = *coverLetter
	}
	path := existing.ResumePath
	if resumePath != nil && *resumePath != "" {
		path = *resumePath
	}
	_, err = s.db.Exec(`
		UPDATE applications SET cover_letter = ?, resume_path = ? WHERE job_id = ?`,
		letter, path, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}
	return s.GetApplication(jobID)
}

// UpsertMatch inserts or replaces match row for a job (used by seed).
func (s *Store) UpsertMatch(m *models.Match) error {
	_, err := s.db.Exec(`
		INSERT INTO matches (job_id, score, track, matched_skills, missing_skills, score_reasons, scored_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			score = excluded.score,
			track = excluded.track,
			matched_skills = excluded.matched_skills,
			missing_skills = excluded.missing_skills,
			score_reasons = excluded.score_reasons,
			scored_at = excluded.scored_at`,
		m.JobID, m.Score, m.Track, m.MatchedSkills, m.MissingSkills, m.ScoreReasons, m.ScoredAt,
	)
	return err
}

// SeedDemo inserts demo pipeline data if seed URLs are missing.
func (s *Store) SeedDemo() (int, error) {
	type seedJob struct {
		source, sourceID, url, title, company, location, description, status string
		salaryMin, salaryMax                                                 int
		score                                                                float64
		track                                                                string
		matched, missing, reasons, cover                                     string
	}

	loc := func(s string) *string { return &s }
	seeds := []seedJob{
		{
			source: "greenhouse", sourceID: "seed-1", url: "https://example.com/jobs/seed-replicate-fs",
			title: "Senior Full-Stack Engineer", company: "Replicate", location: "Remote",
			description: "Build Next.js and NestJS services for AI model APIs. TypeScript, Prisma, PostgreSQL.",
			status:      "discovered", salaryMin: 3000, salaryMax: 4500,
		},
		{
			source: "greenhouse", sourceID: "seed-2", url: "https://example.com/jobs/seed-anysphere-fde",
			title: "Forward Deployed Engineer", company: "Anysphere", location: "Remote",
			description: "Work with customers on agentic workflows, MCP tools, and LLM integrations.",
			status:      "scored", salaryMin: 3500, salaryMax: 5000,
			score: 0.88, track: "fde",
			matched: `["MCP","Agents","LLM"]`, missing: `["On-site travel"]`,
			reasons: "Strong FDE/AI signal from MCP and agentic keywords; remote-friendly.",
			cover:   "I'm excited about Anysphere's agent tooling. I have shipped MCP-based internal agents...",
		},
		{
			source: "lever", sourceID: "seed-3", url: "https://example.com/jobs/seed-polygon-chain",
			title: "Backend Blockchain Engineer", company: "Polygon", location: "Remote",
			description: "EVM smart contracts, Solidity, indexers, and subgraphs for DeFi infrastructure.",
			status:      "queued", salaryMin: 2800, salaryMax: 4200,
			score: 0.82, track: "blockchain",
			matched: `["Solidity","EVM","Indexers"]`, missing: `["Rust"]`,
			reasons: "Heavy protocol/smart-contract focus maps to blockchain resume.",
			cover:   "My DeFi work includes EVM deployments and indexer performance tuning...",
		},
		{
			source: "ashby", sourceID: "seed-4", url: "https://example.com/jobs/seed-vercel-applied",
			title: "Full Stack Engineer", company: "Vercel", location: "Remote",
			description: "Next.js platform features, TypeScript, edge runtimes.",
			status:      "applied", salaryMin: 4000, salaryMax: 6000,
			score: 0.79, track: "fullstack",
			matched: `["Next.js","TypeScript"]`, missing: `["GraphQL"]`,
			reasons: "Core fullstack stack alignment (Next.js, NestJS, TypeScript).",
			cover:   "I have shipped large Next.js/NestJS systems...",
		},
		{
			source: "web3", sourceID: "seed-5", url: "https://example.com/jobs/seed-interview-demo",
			title: "AI Solutions Engineer", company: "Together AI", location: "Remote",
			description: "Customer-facing AI engineering, RAG, prompt systems.",
			status:      "interview", salaryMin: 3200, salaryMax: 4800,
			score: 0.85, track: "fde",
			matched: `["RAG","LLM","Solutions"]`, missing: `["PyTorch research"]`,
			reasons: "Customer/architecture-facing AI role; route to FDE track.",
			cover:   "Happy to continue our conversation about RAG platforms...",
		},
		{
			source: "remoteok", sourceID: "seed-6", url: "https://example.com/jobs/seed-archived",
			title: "Junior WordPress Dev", company: "Example Agency", location: "Onsite",
			description: "Maintain WordPress sites. PHP only.",
			status:      "rejected", salaryMin: 800, salaryMax: 1200,
			score: 0.2, track: "fullstack",
			matched: `[]`, missing: `["WordPress","PHP"]`,
			reasons: "Below compensation floor and weak stack fit.",
			cover:   "",
		},
	}

	created := 0
	now := time.Now().UTC()
	for _, sj := range seeds {
		var existsID int64
		err := s.db.Get(&existsID, `SELECT id FROM jobs WHERE url = ?`, sj.url)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return created, fmt.Errorf("check seed url: %w", err)
		}

		job := &models.Job{
			Source: sj.source, SourceID: sj.sourceID, URL: sj.url,
			Title: sj.title, Company: sj.company, Location: loc(sj.location),
			IsRemote: true, Description: sj.description, Status: sj.status,
			SalaryMin: &sj.salaryMin, SalaryMax: &sj.salaryMax, CreatedAt: now,
		}
		createdJob, err := s.CreateJob(job)
		if err != nil {
			if errors.Is(err, ErrConflict) {
				continue
			}
			return created, err
		}
		created++

		if sj.score > 0 {
			ms, miss, reasons := sj.matched, sj.missing, sj.reasons
			if err := s.UpsertMatch(&models.Match{
				JobID: createdJob.ID, Score: sj.score, Track: sj.track,
				MatchedSkills: &ms, MissingSkills: &miss, ScoreReasons: &reasons,
				ScoredAt: now,
			}); err != nil {
				return created, err
			}
			path, _ := resumes.Path(sj.track)
			letter := sj.cover
			if _, err := s.UpsertApplication(createdJob.ID, &letter, &path); err != nil {
				return created, err
			}
		}
	}
	return created, nil
}
