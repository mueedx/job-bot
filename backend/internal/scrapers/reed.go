package scrapers

import (
	"context"
	"encoding/xml"
	"fmt"

	"net/http"
	"net/url"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// ReedJob is one posting from the Reed API.
type ReedJob struct {
	Title       string `xml:"jobTitle"`
	Company     string `xml:"employerCompany"`
	Location    string `xml:"townOrCity"`
	URL         string `xml:"jobUrl"`
	Excerpt     string `xml:"description"`
	PostedDate  string `xml:"postedDate"`
	RemoteAllow bool   `xml:"remoteWorkingAllowed"`
}

// Reed searches https://www.reed.com/search.xml using REED_API_KEY.
// UK-focused; best suited when your target regions include GB.
type Reed struct {
	Client *http.Client
	APIKey string
}

func (r *Reed) Name() string { return "reed" }

func (r *Reed) Fetch(ctx context.Context) ([]RawJob, error) {
	u, err := url.Parse("https://www.reed.com/search.xml")
	if err != nil {
		return nil, fmt.Errorf("parsing reed URL: %w", err)
	}
	q := u.Query()
	q.Set("apiKey", r.APIKey)
	q.Set("keywords", "developer OR engineer OR programmer OR architect OR blockchain")
	q.Set("pagesize", "250")
	q.Set("orderby", "dateupdated")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building reed request: %w", err)
	}
	req.Header.Set("User-Agent", "job-agent (+https://github.com/mueedx/job-bot)")

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reed returned HTTP %d", resp.StatusCode)
	}

	var raw struct {
		Jobs []ReedJob `xml:"job"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding reed XML: %w", err)
	}

	var out []RawJob
	for _, j := range raw.Jobs {
		if j.Title == "" || j.URL == "" {
			continue
		}

		loc := j.Location

		desc := textutil.HTMLToText(j.Excerpt)
		if desc == "" {
			desc = j.Title
		}

		out = append(out, RawJob{
			Source:      "reed",
			SourceID:    j.URL,
			URL:         j.URL,
			Title:       j.Title,
			Company:     j.Company,
			Location:    loc,
			Description: desc,
			IsRemote:    j.RemoteAllow,
			PublishedAt: parseAnyEpoch(j.PostedDate),
		})
	}

	if len(out) == 0 && len(raw.Jobs) == 0 {
		return nil, fmt.Errorf("reed returned an empty response")
	}

	return out, nil
}
