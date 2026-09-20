package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// Jobicy fetches from https://jobicy.com/api/v2/remote-jobs (or a custom base URL
// in tests). It returns remote jobs with a jobGeo field describing where the role is
// based. Per the API's friendly notice, the source must be credited with a direct
// link to the original job URL (see the registry Note).
type Jobicy struct {
	Client  *http.Client
	BaseURL string // scheme+host without trailing slash or path; defaults to production base
	Geo     string // optional geo slug filter (e.g. "anywhere", "ireland", "uk"); empty = all
}

func (j *Jobicy) Name() string { return "jobicy" }

func (j *Jobicy) Fetch(ctx context.Context) ([]RawJob, error) {
	base := j.BaseURL
	if base == "" {
		base = "https://jobicy.com"
	}
	url := base + "/api/v2/remote-jobs"
	if j.Geo != "" {
		url += "?geo=" + j.Geo
	}
	var payload struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Jobs    []struct {
			ID          int    `json:"id"`
			URL         string `json:"url"`
			JobTitle    string `json:"jobTitle"`
			CompanyName string `json:"companyName"`
			JobGeo      string `json:"jobGeo"`
			JobDesc     string `json:"jobDescription"`
			PubDate     string `json:"pubDate"`
		} `json:"jobs"`
	}

	if err := httpGetJSON(ctx, j.Client, url, &payload); err != nil {
		return nil, err
	}
	if !payload.Success {
		return nil, fmt.Errorf("jobicy: %s", payload.Error)
	}

	var out []RawJob
	for _, row := range payload.Jobs {
		id := fmt.Sprintf("%d", row.ID)
		title := strings.TrimSpace(row.JobTitle)
		company := strings.TrimSpace(row.CompanyName)
		geo := strings.TrimSpace(row.JobGeo)
		desc := textutil.HTMLToText(row.JobDesc)
		if title == "" || company == "" {
			continue
		}
		if desc == "" {
			desc = title
		}

		remote := strings.EqualFold(geo, "anywhere") || strings.Contains(strings.ToLower(geo), "anywhere")

		out = append(out, RawJob{
			Source:      "jobicy",
			SourceID:    id,
			URL:         row.URL,
			Title:       title,
			Company:     company,
			Location:    geo,
			Description: desc,
			IsRemote:    remote || looksRemote(geo),
			PublishedAt: parseISOTime(row.PubDate),
		})
	}
	return out, nil
}
