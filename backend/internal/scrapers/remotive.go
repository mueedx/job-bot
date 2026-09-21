package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// RemotiveConfig holds the fields Remotive jobs expose through the remote-jobs
// endpoint. The schema is a moving target, so we map defensively and validate
// after decoding.
type remotiveJob struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Company     string   `json:"company_name"`
	Location    string   `json:"location"`
	Country     string   `json:"country"`
	Desc        string   `json:"description"`
	Type        string   `json:"job_type"`
	Category    string   `json:"category"`
	Departments []string `json:"departments"`
	ApplyURL    string   `json:"url"`
	HideApply   bool     `json:"hide_apply"`
	UpdatedAt   string   `json:"updated_at"`
}

// Remotive fetches https://remotive.com/api/remote-jobs?category=tech and
// falls back to the homepage-specific endpoint when that returns nothing.
// Per the API Terms the crawler must rate-limit to 4 requests/day; the caller
// is expected to honor that externally. The site itself says remotive.com is
// scrapable provided you keep it to ~4 calls per day total.
type Remotive struct {
	Client *http.Client
	// Key, when non-empty, is unused by this implementation; kept so the
	// registry can present BYO-key sources uniformly. The scrape itself needs
	// no key, only the per-day rate budget.
	Key string
}

func (r *Remotive) Name() string { return "remotive" }

func (r *Remotive) Fetch(ctx context.Context) ([]RawJob, error) {
	// Try category=tech first, then fall back to the homepage-specific endpoint
	// (which lists all remote roles) when the former returns nothing.
	urls := []string{
		"https://remotive.com/api/remote-jobs?category=tech",
		"https://remotive.com/api/remote-jobs?category=homepage",
	}

	var lastErr error
	for i, u := range urls {
		jobs, err := r.fetchURL(ctx, u)
		if err != nil {
			lastErr = err
			continue
		}
		if len(jobs) > 0 {
			return jobs, nil
		}
		// category=tech returning an empty list is a soft signal — try the
		// homepage-specific endpoint as a fallback rather than failing.
		lastErr = fmt.Errorf("remotive returned 0 jobs from %s", u)
		if i == 0 {
			continue
		}
	}

	return nil, lastErr
}

func (r *Remotive) fetchURL(ctx context.Context, rawURL string) ([]RawJob, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	q.Set("limit", "250") // API caps at 250; anything higher is silently clamped.
	parsed.RawQuery = q.Encode()

	var rows []remotiveJob
	if err := httpGetJSON(ctx, r.Client, parsed.String(), &rows); err != nil {
		return nil, err
	}

	var out []RawJob
	for _, j := range rows {
		if j.Title == "" || j.ApplyURL == "" {
			continue
		}
		loc := j.Location
		if loc == "" {
			loc = j.Country
		}
		isRemote := strings.EqualFold(j.Type, "full-time") ||
			strings.Contains(strings.ToLower(loc), "remote") ||
			strings.Contains(strings.ToLower(loc), "anywhere")

		updated := parseAnyEpoch(j.UpdatedAt)
		if updated == nil {
			updated = parseAnyEpoch(j.UpdatedAt)
		}

		desc := textutil.HTMLToText(j.Desc)
		if desc == "" {
			desc = j.Title
		}

		out = append(out, RawJob{
			Source:      "remotive",
			SourceID:    j.ID,
			URL:         j.ApplyURL,
			Title:       j.Title,
			Company:     j.Company,
			Location:    loc,
			Description: desc,
			IsRemote:    isRemote,
			PublishedAt: updated,
		})
	}
	return out, nil
}

// quotaForRemotive returns a human-friendly scrape budget note for this source.
// The Remotive API terms ask for a max of about 4 requests/day.
func quotaForRemotive() string {
	return "max ~4 requests/day (see https://remotive.com/api/#terms)"
}
