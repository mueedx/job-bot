package scrapers

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// RawJob is a normalized posting from any public board source.
type RawJob struct {
	Source      string
	SourceID    string
	URL         string
	Title       string
	Company     string
	Location    string
	Description string
	SalaryMin   *int
	SalaryMax   *int
	IsRemote    bool
	// PublishedAt is when the posting went live, when the source exposes it.
	// Nil means "unknown" — such jobs are never dropped by the age filter.
	PublishedAt *time.Time
}

// Scraper fetches jobs from one source.
type Scraper interface {
	Name() string
	Fetch(ctx context.Context) ([]RawJob, error)
}

// TargetCompanies is the YAML shape for curated board slugs.
type TargetCompanies struct {
	Greenhouse []string `yaml:"greenhouse"`
	Lever      []string `yaml:"lever"`
	Ashby      []string `yaml:"ashby"`
}

// parseISOTime parses an ISO-8601 / RFC3339 timestamp, returning nil on failure.
func parseISOTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			u := t.UTC()
			return &u
		}
	}
	return nil
}

// parseEpochTime parses a Unix epoch expressed in seconds or milliseconds.
func parseEpochTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return nil
	}
	// Heuristic: anything above ~1e12 is milliseconds.
	if n > 1_000_000_000_000 {
		n = n / 1000
	}
	t := time.Unix(n, 0).UTC()
	return &t
}

// parseAnyEpoch handles JSON numbers that may decode as float64.
func parseAnyEpoch(v any) *time.Time {
	switch n := v.(type) {
	case float64:
		return parseEpochTime(strconv.FormatInt(int64(n), 10))
	case int64:
		return parseEpochTime(strconv.FormatInt(n, 10))
	case string:
		return parseEpochTime(n)
	default:
		return nil
	}
}

// firstTime returns the first non-nil timestamp (used for date fallbacks).
func firstTime(candidates ...*time.Time) *time.Time {
	for _, c := range candidates {
		if c != nil {
			return c
		}
	}
	return nil
}
