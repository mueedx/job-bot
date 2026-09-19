package services_test

import (
	"testing"
	"time"

	"github.com/mueedx/job-bot/backend/internal/services"
)

func TestJobMaxAgeDays(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},
		{"0", 0},
		{"7", 7},
		{"30", 30},
		{"abc", 0},
		{"-5", 0},
		{" 14 ", 14},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			t.Setenv("JOB_MAX_AGE_DAYS", c.raw)
			if got := services.JobMaxAgeDays(); got != c.want {
				t.Fatalf("JobMaxAgeDays(%q) = %d, want %d", c.raw, got, c.want)
			}
		})
	}
}

func TestJobAgeCutoffDisabled(t *testing.T) {
	t.Setenv("JOB_MAX_AGE_DAYS", "0")
	if got := services.JobAgeCutoff(); got != nil {
		t.Fatalf("JobAgeCutoff() = %v, want nil when unlimited", got)
	}
}

func TestJobAgeCutoffWindow(t *testing.T) {
	t.Setenv("JOB_MAX_AGE_DAYS", "7")
	got := services.JobAgeCutoff()
	if got == nil {
		t.Fatal("JobAgeCutoff() = nil, want a 7-day window")
	}
	want := time.Now().UTC().AddDate(0, 0, -7)
	if diff := want.Sub(*got); diff > time.Minute || diff < -time.Minute {
		t.Fatalf("JobAgeCutoff() = %v, want about %v", *got, want)
	}
}

func TestTooOld(t *testing.T) {
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -7)
	old := now.AddDate(0, 0, -30)
	recent := now.AddDate(0, 0, -1)

	if !services.TooOld(&old, &cutoff) {
		t.Error("30-day-old posting should be too old for a 7-day window")
	}
	if services.TooOld(&recent, &cutoff) {
		t.Error("1-day-old posting should be kept")
	}
	if services.TooOld(nil, &cutoff) {
		t.Error("posting with unknown date must be kept")
	}
	if services.TooOld(&old, nil) {
		t.Error("disabled window must keep every posting")
	}
}
