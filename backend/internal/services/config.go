package services

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// JobMaxAgeDays returns JOB_MAX_AGE_DAYS: the oldest posting age in days that
// should be kept (0 = no limit). Unset or invalid values mean "no limit".
func JobMaxAgeDays() int {
	raw := strings.TrimSpace(os.Getenv("JOB_MAX_AGE_DAYS"))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// JobAgeCutoff returns the oldest acceptable posting time, or nil when the
// age window is disabled.
func JobAgeCutoff() *time.Time {
	days := JobMaxAgeDays()
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	return &cutoff
}

// TooOld reports whether a posting falls outside the configured age window.
// Postings with an unknown date are never considered too old.
func TooOld(publishedAt *time.Time, cutoff *time.Time) bool {
	if cutoff == nil || publishedAt == nil {
		return false
	}
	return publishedAt.Before(*cutoff)
}
