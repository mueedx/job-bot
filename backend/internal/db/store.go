package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mueedx/job-bot/backend/internal/models"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on unique constraint violations (e.g. duplicate URL).
var ErrConflict = errors.New("conflict")

// Store provides job CRUD and stats queries.
type Store struct {
	db *sqlx.DB
}

// NewStore wraps an open sqlx DB.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// jobColumns is the shared SELECT list for job rows, including the eligibility
// verdict columns so every read returns the stored verdict.
const jobColumns = `id, source, source_id, url, title, company, location,
		       is_remote, is_relocation, salary_min, salary_max,
		       description, posted_at, status, created_at,
		       eligibility, eligibility_rule, eligibility_reason,
		       eligibility_signals, eligibility_applied`

// ListJobs returns jobs ordered by created_at desc, optionally filtered by status.
func (s *Store) ListJobs(status string, limit int) ([]models.Job, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	var jobs []models.Job
	var err error
	if status != "" {
		err = s.db.Select(&jobs, `SELECT `+jobColumns+` FROM jobs WHERE status = ? ORDER BY created_at DESC LIMIT ?`, status, limit)
	} else {
		err = s.db.Select(&jobs, `SELECT `+jobColumns+` FROM jobs ORDER BY created_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	if jobs == nil {
		jobs = []models.Job{}
	}
	return jobs, nil
}

// GetJob returns a job by id.
func (s *Store) GetJob(id int64) (*models.Job, error) {
	var job models.Job
	err := s.db.Get(&job, `SELECT `+jobColumns+` FROM jobs WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &job, nil
}

// CreateJob inserts a job and returns the created row.
func (s *Store) CreateJob(job *models.Job) (*models.Job, error) {
	if job.Status == "" {
		job.Status = "discovered"
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}

	res, err := s.db.Exec(`
		INSERT INTO jobs (
			source, source_id, url, title, company, location,
			is_remote, is_relocation, salary_min, salary_max,
			description, posted_at, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.Source, job.SourceID, job.URL, job.Title, job.Company, job.Location,
		job.IsRemote, job.IsRelocation, job.SalaryMin, job.SalaryMax,
		job.Description, job.PostedAt, job.Status, job.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create job: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return s.GetJob(id)
}

// UpdateJob applies non-nil fields from patch onto the existing job.
func (s *Store) UpdateJob(id int64, patch *models.JobPatch) (*models.Job, error) {
	job, err := s.GetJob(id)
	if err != nil {
		return nil, err
	}

	if patch.Status != nil {
		job.Status = *patch.Status
	}
	if patch.Location != nil {
		job.Location = patch.Location
	}
	if patch.IsRemote != nil {
		job.IsRemote = *patch.IsRemote
	}
	if patch.IsRelocation != nil {
		job.IsRelocation = *patch.IsRelocation
	}
	if patch.SalaryMin != nil {
		job.SalaryMin = patch.SalaryMin
	}
	if patch.SalaryMax != nil {
		job.SalaryMax = patch.SalaryMax
	}
	if patch.Title != nil {
		job.Title = *patch.Title
	}
	if patch.Description != nil {
		job.Description = *patch.Description
	}

	_, err = s.db.Exec(`
		UPDATE jobs SET
			status = ?, location = ?, is_remote = ?, is_relocation = ?,
			salary_min = ?, salary_max = ?, title = ?, description = ?
		WHERE id = ?`,
		job.Status, job.Location, job.IsRemote, job.IsRelocation,
		job.SalaryMin, job.SalaryMax, job.Title, job.Description, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	return s.GetJob(id)
}

// GetStats aggregates job and application counts.
func (s *Store) GetStats() (*models.Stats, error) {
	byStatus := map[string]int{}
	rows, err := s.db.Queryx(`SELECT status, COUNT(*) AS cnt FROM jobs GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("stats by status: %w", err)
	}
	defer rows.Close()

	totalJobs := 0
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		byStatus[status] = cnt
		totalJobs += cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var totalApplications int
	if err := s.db.Get(&totalApplications, `SELECT COUNT(*) FROM applications`); err != nil {
		return nil, fmt.Errorf("count applications: %w", err)
	}

	return &models.Stats{
		TotalJobs:         totalJobs,
		TotalApplications: totalApplications,
		Applied:           byStatus["applied"],
		Interviews:        byStatus["interview"],
		ByStatus:          byStatus,
	}, nil
}

func isUniqueViolation(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed")
}

// SetApplicationQA updates custom_qa JSON for a job's application.
func (s *Store) SetApplicationQA(jobID int64, qa string) error {
	_, err := s.db.Exec(`UPDATE applications SET custom_qa = ? WHERE job_id = ?`, qa, jobID)
	return err
}

// ClearEligibilityApplied hands ownership of a job's status back to the
// operator. Called whenever the operator sets a status (board drag, approve,
// discard, Telegram action) so a later rules re-check never moves a card the
// operator put somewhere on purpose.
func (s *Store) ClearEligibilityApplied(jobID int64) error {
	_, err := s.db.Exec(`UPDATE jobs SET eligibility_applied = 0 WHERE id = ?`, jobID)
	if err != nil {
		return fmt.Errorf("clear eligibility applied: %w", err)
	}
	return nil
}

// JobEligibility is the stored eligibility verdict for one job.
type JobEligibility struct {
	Status  string
	Rule    string
	Reason  string
	Signals string
	Applied bool
}

// SetJobEligibility stores the verdict for a job. Applied records whether the
// engine introduced the job's current status (see EligibilityApplied).
func (s *Store) SetJobEligibility(jobID int64, e JobEligibility) error {
	_, err := s.db.Exec(`
		UPDATE jobs SET
			eligibility = ?, eligibility_rule = ?, eligibility_reason = ?,
			eligibility_signals = ?, eligibility_applied = ?
		WHERE id = ?`,
		e.Status, e.Rule, e.Reason, e.Signals, e.Applied, jobID)
	if err != nil {
		return fmt.Errorf("set job eligibility: %w", err)
	}
	return nil
}

// CountJobsCreatedToday returns jobs created since UTC midnight.
func (s *Store) CountJobsByStatusToday(statuses ...string) (int, error) {
	if len(statuses) == 0 {
		return 0, nil
	}
	q, args, err := sqlx.In(`
		SELECT COUNT(*) FROM jobs
		WHERE status IN (?) AND date(created_at) = date('now')`, statuses)
	if err != nil {
		return 0, err
	}
	q = s.db.Rebind(q)
	var n int
	if err := s.db.Get(&n, q, args...); err != nil {
		return 0, err
	}
	return n, nil
}
