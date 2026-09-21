package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// AdzunaJob is one posting from the Adzuna API v1.
type AdzunaJob struct {
	Title     string `json:"title"`
	Company   string `json:"company"`
	Location  string `json:"location"`
	URL       string `json:"redirect_url"`
	Description string `json:"description"`
	Created   string `json:"created"`
	Country   string `json:"country"`
}

// Adzuna queries https://api.adzuna.com/v1/api/jobs/search/{country} using
// ADZUNA_APP_ID + ADZUNA_APP_KEY. Country codes are Adzuna's own (e.g. "ie",
// "gb", "au", "nz").
type Adzuna struct {
	Client *http.Client
	AppID  string
	AppKey string
	Country string
}

func (a *Adzuna) Name() string { return "adzuna" }

func (a *Adzuna) Fetch(ctx context.Context) ([]RawJob, error) {
	path := fmt.Sprintf("https://api.adzuna.com/v1/api/jobs/search/%s", a.Country)
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing Adzuna URL: %w", err)
	}
	q := u.Query()
	q.Set("app_id", a.AppID)
	q.Set("app_key", a.AppKey)
	q.Set("results_per_page", "250")
	q.Set("sort_by", "date")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building adzuna request: %w", err)
	}
	req.Header.Set("User-Agent", "job-agent (+https://github.com/mueedx/job-bot)")

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("adzuna: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adzuna returned HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Results []AdzunaJob `json:"results"`
	}
	if err := httpGetJSON(ctx, a.Client, u.String(), &wrapper); err != nil {
		return nil, err
	}

	var out []RawJob
	for _, j := range wrapper.Results {
		if j.Title == "" || j.URL == "" {
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
			Source:      "adzuna",
			SourceID:    j.URL,
			URL:         j.URL,
			Title:       j.Title,
			Company:     j.Company,
			Location:    loc,
			Description: desc,
			IsRemote:    strings.Contains(strings.ToLower(loc), "remote"),
			PublishedAt: parseAnyEpoch(j.Created),
		})
	}

	if len(out) == 0 && len(wrapper.Results) == 0 {
		return nil, fmt.Errorf("adzuna returned an empty response")
	}

	return out, nil
}
