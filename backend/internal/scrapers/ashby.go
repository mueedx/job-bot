package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// Ashby scrapes public Ashby job board JSON.
type Ashby struct {
	Client *http.Client
	Slugs  []string
}

func (a *Ashby) Name() string { return "ashby" }

func (a *Ashby) Fetch(ctx context.Context) ([]RawJob, error) {
	var out []RawJob
	var errs []string
	for _, slug := range a.Slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		var payload struct {
			Jobs []struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				JobURL      string `json:"jobUrl"`
				Location    string `json:"location"`
				Description string `json:"descriptionHtml"`
				IsRemote    bool   `json:"isRemote"`
				PublishedAt string `json:"publishedAt"`
			} `json:"jobs"`
		}
		url := fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s?includeCompensation=true", slug)
		if err := httpGetJSON(ctx, a.Client, url, &payload); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", slug, err))
			continue
		}
		for _, j := range payload.Jobs {
			desc := textutil.HTMLToText(j.Description)
			if desc == "" {
				desc = j.Title
			}
			out = append(out, RawJob{
				Source:      "ashby",
				SourceID:    j.ID,
				URL:         j.JobURL,
				Title:       j.Title,
				Company:     slug,
				Location:    j.Location,
				Description: desc,
				IsRemote:    j.IsRemote || looksRemote(j.Location),
				PublishedAt: parseISOTime(j.PublishedAt),
			})
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return out, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}
