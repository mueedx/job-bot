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

func (c *CryptoJobs) Name() string { return "cryptojobs" }

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
