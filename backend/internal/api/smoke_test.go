package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mueedx/job-bot/backend/internal/api"
	"github.com/mueedx/job-bot/backend/internal/db"
)

func TestPhase1Smoke(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if _, err := os.Stat(filepath.Join(dir, "jobs.db")); err != nil {
		t.Fatalf("jobs.db missing: %v", err)
	}

	var tables []string
	if err := sqlDB.Select(&tables, `SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`); err != nil {
		t.Fatalf("list tables: %v", err)
	}
	wantTables := map[string]bool{
		"jobs": true, "matches": true, "applications": true, "submission_logs": true,
	}
	for _, name := range tables {
		delete(wantTables, name)
	}
	if len(wantTables) != 0 {
		t.Fatalf("missing tables: %v (have %v)", wantTables, tables)
	}

	srv := &api.Server{Store: db.NewStore(sqlDB)}
	handler := api.NewRouter(srv)

	// health
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health: %d %s", rr.Code, rr.Body.String())
	}

	// docs + openapi
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("docs: %d %s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("openapi:")) {
		t.Fatalf("openapi: %d %s", rr.Code, rr.Body.String())
	}

	// stats empty
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", rr.Code, rr.Body.String())
	}

	// create job
	body := []byte(`{
		"source":"greenhouse","source_id":"demo-1",
		"url":"https://example.com/jobs/demo-go-1",
		"title":"Full Stack Engineer","company":"Example Co",
		"description":"Build things with Next.js and NestJS"
	}`)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, ok := created["id"].(float64)
	if !ok || id < 1 {
		t.Fatalf("bad id: %#v", created["id"])
	}

	// duplicate → 409
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body)))
	if rr.Code != http.StatusConflict {
		t.Fatalf("dup want 409 got %d %s", rr.Code, rr.Body.String())
	}

	// apply stub → 501
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/apply/1", nil)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("apply want 501 got %d %s", rr.Code, rr.Body.String())
	}

	// stats after create
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	var stats map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats["total_jobs"].(float64) < 1 {
		t.Fatalf("expected total_jobs >= 1: %#v", stats)
	}
	if _, ok := stats["daily_limit"]; !ok {
		t.Fatalf("expected daily_limit in stats: %#v", stats)
	}

	// seed + detail + application + approve honesty + discard
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/dev/seed", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("seed: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/jobs?enrich=1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("enrich list: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/jobs/1/detail", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("detail: %d %s", rr.Code, rr.Body.String())
	}

	appBody := []byte(`{"cover_letter":"Hello","track":"fullstack"}`)
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/jobs/1/application", bytes.NewReader(appBody))
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("upsert app: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/jobs/1/approve", nil))
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("approve want 501 got %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/jobs/1/discard", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("discard: %d %s", rr.Code, rr.Body.String())
	}
}
