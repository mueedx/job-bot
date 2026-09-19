package scrapers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// RemoteOK scrapes https://remoteok.com/api
type RemoteOK struct {
	Client *http.Client
}

func (r *RemoteOK) Name() string { return "remoteok" }

func (r *RemoteOK) Fetch(ctx context.Context) ([]RawJob, error) {
	var rows []map[string]any
	if err := httpGetJSON(ctx, r.Client, "https://remoteok.com/api", &rows); err != nil {
		return nil, err
	}
	var out []RawJob
	for _, row := range rows {
		// First element is often metadata without id.
		id, _ := row["id"].(string)
		if id == "" {
			if n, ok := row["id"].(float64); ok {
				id = fmt.Sprintf("%.0f", n)
			}
		}
		if id == "" || id == "0" {
			continue
		}
		title, _ := row["position"].(string)
		if title == "" {
			title, _ = row["title"].(string)
		}
		company, _ := row["company"].(string)
		url, _ := row["url"].(string)
		if url == "" {
			url, _ = row["apply_url"].(string)
		}
		desc, _ := row["description"].(string)
		loc, _ := row["location"].(string)
		tags := fmt.Sprint(row["tags"])
		blob := strings.ToLower(title + " " + tags + " " + desc)
		if !strings.Contains(blob, "engineer") &&
			!strings.Contains(blob, "developer") &&
			!strings.Contains(blob, "full stack") &&
			!strings.Contains(blob, "frontend") &&
			!strings.Contains(blob, "backend") &&
			!strings.Contains(blob, "software") &&
			!strings.Contains(blob, "blockchain") &&
			!strings.Contains(blob, "ai ") {
			continue
		}
		if url == "" {
			continue
		}
		out = append(out, RawJob{
			Source:      "remoteok",
			SourceID:    id,
			URL:         url,
			Title:       title,
			Company:     company,
			Location:    loc,
			Description: textutil.HTMLToText(desc),
			IsRemote:    true,
			PublishedAt: parseAnyEpoch(row["epoch"]),
		})
	}
	return out, nil
}
