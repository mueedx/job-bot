package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func httpGetJSON(ctx context.Context, client *http.Client, url string, dest any) error {
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "job-agent/0.2 (+https://github.com/mueedx/job-bot)")
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d for %s: %s", res.StatusCode, url, truncate(string(body), 200))
	}
	return json.Unmarshal(body, dest)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func looksRemote(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "remote") || strings.Contains(l, "anywhere") || strings.Contains(l, "distributed")
}

func intPtr(v int) *int { return &v }
