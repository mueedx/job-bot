package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// Lever scrapes public Lever postings JSON.
type Lever struct {
	Client *http.Client
	Slugs  []string
}

func (l *Lever) Name() string { return "lever" }

func (l *Lever) Fetch(ctx context.Context) ([]RawJob, error) {
	var out []RawJob
	var errs []string
	for _, slug := range l.Slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		var jobs []struct {
			ID         string `json:"id"`
			Text       string `json:"text"`
			HostedURL  string `json:"hostedUrl"`
			CreatedAt  int64  `json:"createdAt"`
			Categories struct {
				Location string `json:"location"`
				Team     string `json:"team"`
			} `json:"categories"`
			DescriptionPlain string `json:"descriptionPlain"`
			Description      string `json:"description"`
		}
		url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", slug)
		if err := httpGetJSON(ctx, l.Client, url, &jobs); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", slug, err))
			continue
		}
		for _, j := range jobs {
			desc := j.DescriptionPlain
			if desc == "" {
				desc = textutil.HTMLToText(j.Description)
			}
			if desc == "" {
				desc = j.Text
			}
			loc := j.Categories.Location
			out = append(out, RawJob{
				Source:      "lever",
				SourceID:    j.ID,
				URL:         j.HostedURL,
				Title:       j.Text,
				Company:     slug,
				Location:    loc,
				Description: desc,
				IsRemote:    looksRemote(loc + " " + j.Text),
				PublishedAt: parseAnyEpoch(j.CreatedAt),
			})
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return out, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}
