package scrapers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mueedx/job-bot/backend/internal/textutil"
)

// CryptoJobs fetches crypto-tagged remote roles via RemoteOK tag filter
// (public feed; soft-fail friendly).
type CryptoJobs struct {
	Client *http.Client
}

// Name is "web3" — not "cryptojobs" — because that is the value this scraper has
// always written into jobs.source, and the dashboard keys paywall notes off it.
func (c *CryptoJobs) Name() string { return "web3" }

func (c *CryptoJobs) Fetch(ctx context.Context) ([]RawJob, error) {
	var rows []map[string]any
	if err := httpGetJSON(ctx, c.Client, "https://remoteok.com/api?tag=crypto", &rows); err != nil {
		return nil, err
	}
	var out []RawJob
	for _, row := range rows {
		id := fmt.Sprint(row["id"])
		if id == "" || id == "0" || id == "<nil>" {
			continue
		}
		title, _ := row["position"].(string)
		company, _ := row["company"].(string)
		url, _ := row["url"].(string)
		desc, _ := row["description"].(string)
		loc, _ := row["location"].(string)
		if url == "" || title == "" {
			continue
		}
		out = append(out, RawJob{
			Source:      "web3",
			SourceID:    "crypto-" + id,
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
