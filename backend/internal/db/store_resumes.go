package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/mueedx/job-bot/backend/internal/models"
)

// Resume profile persistence: one row per resume file describing what the AI
// extracted from it. Callers key on the stored path ("resumes/x.pdf") and use
// ContentHash to decide whether the file changed since extraction.

// GetResumeProfile returns the profile for a resume path, or nil when none
// has been extracted yet.
func (s *Store) GetResumeProfile(resumePath string) (*models.ResumeProfile, error) {
	var p models.ResumeProfile
	err := s.db.Get(&p, `SELECT resume_path, content_hash, track, skills, keywords, seniority, summary, extracted_at
		FROM resume_profiles WHERE resume_path = ?`, resumePath)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		var rows []models.ResumeProfile
		qErr := s.db.Select(&rows, `SELECT resume_path, content_hash, track, skills, keywords, seniority, summary, extracted_at
			FROM resume_profiles WHERE resume_path = ?`, resumePath)
		if qErr != nil || len(rows) == 0 {
			return nil, nil
		}
		p = rows[0]
	}
	return &p, nil
}

// UpsertResumeProfile stores (or replaces) the extracted profile for a path.
func (s *Store) UpsertResumeProfile(p *models.ResumeProfile) error {
	if p.ExtractedAt.IsZero() {
		p.ExtractedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`INSERT INTO resume_profiles
		(resume_path, content_hash, track, skills, keywords, seniority, summary, extracted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(resume_path) DO UPDATE SET
			content_hash = excluded.content_hash,
			track = excluded.track,
			skills = excluded.skills,
			keywords = excluded.keywords,
			seniority = excluded.seniority,
			summary = excluded.summary,
			extracted_at = excluded.extracted_at`,
		p.ResumePath, p.ContentHash, p.Track, p.Skills, p.Keywords, p.Seniority, p.Summary, p.ExtractedAt)
	if err != nil {
		return fmt.Errorf("upsert resume profile: %w", err)
	}
	return nil
}

// ListResumeProfiles returns every extracted profile.
func (s *Store) ListResumeProfiles() ([]models.ResumeProfile, error) {
	var out []models.ResumeProfile
	err := s.db.Select(&out, `SELECT resume_path, content_hash, track, skills, keywords, seniority, summary, extracted_at
		FROM resume_profiles ORDER BY resume_path`)
	if err != nil {
		return nil, fmt.Errorf("list resume profiles: %w", err)
	}
	return out, nil
}
