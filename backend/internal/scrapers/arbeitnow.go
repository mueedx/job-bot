package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// Arbeitnow fetches from https://www.arbeitnow.com/api/job-board-api (or a custom
// base URL in tests). It returns mostly German/European listings (the API has no
// country filter), with a boolean remote field and a free-text location. Treat it
// as a European feed: the registry Note explains the geographic skew so users
// targeting other regions can switch it off.
type Arbeitnow struct {
	Client  *http.Client
	BaseURL string // scheme+host without trailing slash or path; defaults to production base
}

func (a *Arbeitnow) Name() string { return "arbeitnow" }

func (a *Arbeitnow) Fetch(ctx context.Context) ([]RawJob, error) {
	base := a.BaseURL
	if base == "" {
		base = "https://www.arbeitnow.com"
	}
	url := base + "/api/job-board-api"
	var payload struct {
		Data []struct {
			Slug        string   `json:"slug"`
			CompanyName string   `json:"company_name"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Remote      bool     `json:"remote"`
			URL         string   `json:"url"`
			Tags        []string `json:"tags"`
			Location    string   `json:"location"`
			CreatedAt   int64    `json:"created_at"`
		} `json:"data"`
	}

	if err := httpGetJSON(ctx, a.Client, url, &payload); err != nil {
		return nil, err
	}

	var out []RawJob
	for _, row := range payload.Data {
		title := strings.TrimSpace(row.Title)
		company := strings.TrimSpace(row.CompanyName)
		loc := strings.TrimSpace(row.Location)
		desc := textutil.HTMLToText(row.Description)
		if title == "" || company == "" {
			continue
		}
		if desc == "" {
			desc = title
		}

		remote := row.Remote || looksRemote(loc)

		out = append(out, RawJob{
			Source:      "arbeitnow",
			SourceID:    row.Slug,
			URL:         row.URL,
			Title:       title,
			Company:     company,
			Location:    loc,
			Description: desc,
			IsRemote:    remote,
			PublishedAt: parseEpochTime(fmt.Sprintf("%d", row.CreatedAt)),
		})
	}
	return out, nil
}
