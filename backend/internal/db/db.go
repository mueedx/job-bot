package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/mueedx/job-bot/backend/internal/textutil"
	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database at dataDir/jobs.db and runs migrations.
func Open(dataDir string) (*sqlx.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "jobs.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", dbPath)

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// Migrate applies schema.sql (CREATE TABLE IF NOT EXISTS) plus additive
// migrations so databases created by older builds gain newer columns.
func Migrate(db *sqlx.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	if err := addColumnIfMissing(db, "jobs", "posted_at", "DATETIME"); err != nil {
		return err
	}
	// Eligibility verdict columns (services.EligibilityRules).
	for _, col := range []struct{ name, decl string }{
		{"eligibility", "TEXT"},
		{"eligibility_rule", "TEXT"},
		{"eligibility_reason", "TEXT"},
		{"eligibility_signals", "TEXT"},
		{"eligibility_applied", "BOOLEAN DEFAULT 0"},
	} {
		if err := addColumnIfMissing(db, "jobs", col.name, col.decl); err != nil {
			return err
		}
	}
	if err := repairDescriptions(db); err != nil {
		return err
	}
	return nil
}

// repairDescriptions is a one-time data migration for rows stored before
// descriptions were properly cleaned: it re-flattens any that still contain
// raw HTML tags or HTML entities. Plain-text rows are left byte-for-byte
// unchanged (HTMLToText is idempotent), so the scan is effectively a no-op
// on every startup after the first.
func repairDescriptions(db *sqlx.DB) error {
	type row struct {
		ID          int64  `db:"id"`
		Description string `db:"description"`
	}
	var rows []row
	if err := db.Select(&rows, `
		SELECT id, description FROM jobs
		WHERE description LIKE '%<%' OR description LIKE '%&%;%'`); err != nil {
		return fmt.Errorf("scan descriptions to repair: %w", err)
	}
	for _, r := range rows {
		cleaned := textutil.HTMLToText(r.Description)
		if cleaned == r.Description {
			continue
		}
		if _, err := db.Exec(`UPDATE jobs SET description = ? WHERE id = ?`, cleaned, r.ID); err != nil {
			return fmt.Errorf("repair description %d: %w", r.ID, err)
		}
	}
	return nil
}

// addColumnIfMissing adds a column when an existing table lacks it.
// table/column/decl come from internal constants, never user input.
func addColumnIfMissing(db *sqlx.DB, table, column, decl string) error {
	var count int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?`, table)
	if err := db.Get(&count, q, column); err != nil {
		return fmt.Errorf("check column %s.%s: %w", table, column, err)
	}
	if count > 0 {
		return nil
	}
	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, decl)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}
