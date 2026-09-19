package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
	_ "modernc.org/sqlite"
)

// openStore opens a migrated store in a temp dir.
func openStore(t *testing.T) (*db.Store, *sqlx.DB) {
	t.Helper()
	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db.NewStore(sqlDB), sqlDB
}

func createJob(t *testing.T, store *db.Store, url string, posted *time.Time) *models.Job {
	t.Helper()
	job, err := store.CreateJob(&models.Job{
		Source: "test", SourceID: url, URL: url,
		Title: "Engineer", Company: "Acme", Description: "desc",
		PostedAt: posted,
	})
	if err != nil {
		t.Fatalf("CreateJob(%s): %v", url, err)
	}
	return job
}

// TestMigrateAddsPostedAtToLegacyDB simulates a database created before the
// posted_at column existed and asserts the additive migration adds it in place.
func TestMigrateAddsPostedAtToLegacyDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.db")

	legacy, err := sqlx.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if _, err := legacy.Exec(`CREATE TABLE jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL, source_id TEXT NOT NULL, url TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL, company TEXT NOT NULL, location TEXT,
		is_remote BOOLEAN DEFAULT 1, is_relocation BOOLEAN DEFAULT 0,
		salary_min INTEGER, salary_max INTEGER, description TEXT NOT NULL,
		status TEXT DEFAULT 'discovered', created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := legacy.Exec(`INSERT INTO jobs (source, source_id, url, title, company, description)
		VALUES ('legacy', '1', 'https://example.com/legacy', 'Old Job', 'Acme', 'desc')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatalf("db.Open on legacy file: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	var count int
	if err := sqlDB.Get(&count, `SELECT COUNT(*) FROM pragma_table_info('jobs') WHERE name = 'posted_at'`); err != nil {
		t.Fatalf("check posted_at: %v", err)
	}
	if count != 1 {
		t.Fatalf("posted_at column count = %d, want 1", count)
	}

	// The pre-existing row survives and reads back with an unknown date.
	job, err := db.NewStore(sqlDB).GetJob(1)
	if err != nil {
		t.Fatalf("GetJob on legacy row: %v", err)
	}
	if job.PostedAt != nil {
		t.Fatalf("legacy posted_at = %v, want nil", job.PostedAt)
	}
}

func TestCreateJobPersistsPostedAt(t *testing.T) {
	store, _ := openStore(t)
	posted := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)

	job := createJob(t, store, "https://example.com/dated", &posted)
	if job.PostedAt == nil {
		t.Fatal("posted_at = nil, want the value we passed in")
	}
	if job.PostedAt.Unix() != posted.Unix() {
		t.Fatalf("posted_at = %v, want %v", job.PostedAt, posted)
	}

	// Read path (GetJob) must round-trip the value too.
	again, err := store.GetJob(job.ID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if again.PostedAt == nil || again.PostedAt.Unix() != posted.Unix() {
		t.Fatalf("GetJob posted_at = %v, want %v", again.PostedAt, posted)
	}
}

func TestListJobsEnrichedAgeFilter(t *testing.T) {
	store, _ := openStore(t)
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	recent := now.AddDate(0, 0, -1)

	createJob(t, store, "https://example.com/old", &old)
	createJob(t, store, "https://example.com/recent", &recent)
	createJob(t, store, "https://example.com/unknown", nil)

	cutoff := now.AddDate(0, 0, -7)

	// With the age window active, only recent + unknown-date postings remain.
	items, err := store.ListJobsEnriched("", 50, &cutoff)
	if err != nil {
		t.Fatalf("ListJobsEnriched with cutoff: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d jobs, want 2 (recent + unknown date)", len(items))
	}
	for _, it := range items {
		if it.URL == "https://example.com/old" {
			t.Fatalf("30-day-old posting should be hidden by a 7-day window")
		}
	}

	// With the window disabled, everything is visible.
	all, err := store.ListJobsEnriched("", 50, nil)
	if err != nil {
		t.Fatalf("ListJobsEnriched without cutoff: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d jobs, want 3 when the window is disabled", len(all))
	}

	// Status filter combines with the age window.
	items, err = store.ListJobsEnriched("discovered", 50, &cutoff)
	if err != nil {
		t.Fatalf("ListJobsEnriched with status + cutoff: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d discovered jobs, want 2", len(items))
	}
}
