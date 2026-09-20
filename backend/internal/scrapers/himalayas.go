package scrapers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// himalayasJob is the shape returned by the Himalayas jobs API v1.
type himalayasJob struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Company     string `json:"company_name"`
	Location    string `json:"location"`
	Country     string `json:"country"`
	Description string `json:"description"`
	ApplyURL    string `json:"apply_url"`
	Hireable    bool   `json:"is_hireable"`
	PublishedAt string `json:"published_at"`
	JobType     string `json:"job_type"`
}

// Himalayas fetches remote job postings from https://jobs.himalayas.app/api/v1/jobs.
// No authentication required; public JSON listing.
type Himalayas struct {
	Client *http.Client
}

func (h *Himalayas) Name() string { return "himalayas" }

func (h *Himalayas) Fetch(ctx context.Context) ([]RawJob, error) {
	var rows []himalayasJob
	if err := httpGetJSON(ctx, h.Client, "https://jobs.himalayas.app/api/v1/jobs", &rows); err != nil {
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

		desc := textutil.HTMLToText(j.Description)
		if desc == "" {
			desc = j.Title
		}

		out = append(out, RawJob{
			Source:      "himalayas",
			SourceID:    j.ID,
			URL:         j.ApplyURL,
			Title:       j.Title,
			Company:     j.Company,
			Location:    loc,
			Description: desc,
			IsRemote:    true,
			PublishedAt: parseAnyEpoch(j.PublishedAt),
		})
	}

	if len(out) == 0 && len(rows) == 0 {
		return nil, fmt.Errorf("himalayas returned an empty response")
	}

	return out, nil
}
