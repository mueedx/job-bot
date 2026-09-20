package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// CareerjetJob is one posting from the Careerjet search API.
type CareerjetJob struct {
	Title    string `json:"title"`
	Company  string `json:"company"`
	Location string `json:"location"`
	URL      string `json:"redirect_url"`
	Excerpt  string `json:"description"`
	Date     string `json:"date"`
	Country  string `json:"countrycode"`
	Remote   *bool  `json:"remote"`
}

// Careerjet keys listings by country code from https://www.careerjet.com/api/
// search. Requires CAREERJET_API_KEY. Public documentation at
// https://www.careerjet.com/api/docs.
type Careerjet struct {
	Client *http.Client
	APIKey string
	Country string // ISO 3166-1 alpha-2, e.g. "ie", "gb", "pk", "au"
}

func (c *Careerjet) Name() string { return "careerjet" }

func (c *Careerjet) Fetch(ctx context.Context) ([]RawJob, error) {
	base, err := url.Parse("https://www.careerjet.com/api/search")
	if err != nil {
		return nil, fmt.Errorf("parsing careerjet base URL: %w", err)
	}
	q := base.Query()
	q.Set("key", c.APIKey)
	q.Set("cid", c.Country)
	q.Set("sort", "date")
	q.Set("pagesize", "250")
	base.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building careerjet request: %w", err)
	}
	req.Header.Set("User-Agent", "job-agent (+https://github.com/mueedx/job-bot)")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("careerjet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("careerjet returned HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Jobs []CareerjetJob `json:"jobs"`
	}
	if err := httpGetJSON(ctx, c.Client, base.String(), &wrapper); err != nil {
		return nil, err
	}

	var out []RawJob
	for _, j := range wrapper.Jobs {
		if j.Title == "" || j.URL == "" {
			continue
		}

		loc := j.Location
		if loc == "" {
			loc = j.Country
		}

		isRemote := false
		if j.Remote != nil {
			isRemote = *j.Remote
		} else if strings.Contains(strings.ToLower(loc), "remote") ||
			strings.Contains(strings.ToLower(loc), "anywhere") ||
			strings.Contains(strings.ToLower(loc), "distributed") {
			isRemote = true
		}

		desc := textutil.HTMLToText(j.Excerpt)
		if desc == "" {
			desc = j.Title
		}

		out = append(out, RawJob{
			Source:      "careerjet",
			SourceID:    j.URL,
			URL:         j.URL,
			Title:       j.Title,
			Company:     j.Company,
			Location:    loc,
			Description: desc,
			IsRemote:    isRemote,
			PublishedAt: parseAnyEpoch(j.Date),
		})
	}

	if len(out) == 0 && len(wrapper.Jobs) == 0 {
		return nil, fmt.Errorf("careerjet returned an empty response")
	}

	return out, nil
}
