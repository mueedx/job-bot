package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// Greenhouse scrapes public Greenhouse board APIs.
type Greenhouse struct {
	Client *http.Client
	Slugs  []string
}

func (g *Greenhouse) Name() string { return "greenhouse" }

func (g *Greenhouse) Fetch(ctx context.Context) ([]RawJob, error) {
	var out []RawJob
	var errs []string
	for _, slug := range g.Slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		var payload struct {
			Jobs []struct {
				ID       int64  `json:"id"`
				Title    string `json:"title"`
				Absolute string `json:"absolute_url"`
				Location struct {
					Name string `json:"name"`
				} `json:"location"`
				Content        string `json:"content"`
				FirstPublished string `json:"first_published"`
				UpdatedAt      string `json:"updated_at"`
			} `json:"jobs"`
		}
		url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", slug)
		if err := httpGetJSON(ctx, g.Client, url, &payload); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", slug, err))
			continue
		}
		company := slug
		for _, j := range payload.Jobs {
			loc := j.Location.Name
			desc := textutil.HTMLToText(j.Content)
			if desc == "" {
				desc = j.Title
			}
			out = append(out, RawJob{
				Source:      "greenhouse",
				SourceID:    fmt.Sprintf("%s-%d", slug, j.ID),
				URL:         j.Absolute,
				Title:       j.Title,
				Company:     company,
				Location:    loc,
				Description: desc,
				IsRemote:    looksRemote(loc + " " + j.Title),
				PublishedAt: firstTime(parseISOTime(j.FirstPublished), parseISOTime(j.UpdatedAt)),
			})
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return out, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}
