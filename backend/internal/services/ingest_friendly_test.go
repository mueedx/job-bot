package services

import (
	"errors"
	"strings"
	"testing"
)

func TestFriendlySourceError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("greenhouse GET: 404 not found"), "not found"},
		{errors.New("context deadline exceeded"), "timed out"},
		{errors.New("connection refused"), "could not reach"},
		{errors.New("HTTP 429 too many requests"), "rate limited"},
	}
	for _, tc := range cases {
		got := friendlySourceError("greenhouse", tc.err)
		if !strings.Contains(strings.ToLower(got), tc.want) {
			t.Fatalf("friendly(%v) = %q, want substring %q", tc.err, got, tc.want)
		}
		if strings.Contains(got, "greenhouse GET") {
			t.Fatalf("should not dump raw scraper prefix: %q", got)
		}
	}
}
