package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mueedx/job-bot/backend/internal/api"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/services"
)

func TestIngestAsyncSmoke(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	// Empty board lists — RemoteOK/CryptoJobs still hit the network; keep timeout generous.
	targets := []byte("greenhouse: []\nlever: []\nashby: []\n")
	if err := os.WriteFile(filepath.Join(dataDir, "target_companies.yaml"), targets, 0o644); err != nil {
		t.Fatalf("write targets: %v", err)
	}

	ingestor := &services.Ingestor{
		Store:   db.NewStore(sqlDB),
		DataDir: dataDir,
		Client:  &http.Client{Timeout: 15 * time.Second},
	}
	srv := &api.Server{Store: db.NewStore(sqlDB), Ingestor: ingestor}
	handler := api.NewRouter(srv)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/ingest/run", nil))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("ingest run want 202 got %d %s", rr.Code, rr.Body.String())
	}
	var st map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode 202 body: %v", err)
	}
	if st["running"] != true {
		t.Fatalf("expected running=true on 202: %#v", st)
	}
	if _, ok := st["logs"]; !ok {
		t.Fatalf("expected logs on status: %#v", st)
	}
	if _, ok := st["phase"]; !ok {
		t.Fatalf("expected phase on status: %#v", st)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/ingest/run", nil))
	if rr.Code != http.StatusConflict {
		t.Fatalf("second run want 409 got %d %s", rr.Code, rr.Body.String())
	}
	var conflict map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &conflict); err != nil {
		t.Fatalf("decode 409: %v", err)
	}
	if conflict["detail"] != "A search is already running." {
		t.Fatalf("unexpected 409 detail: %#v", conflict)
	}

	deadline := time.Now().Add(90 * time.Second)
	var last map[string]any
	for time.Now().Before(deadline) {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/ingest/status", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status: %d %s", rr.Code, rr.Body.String())
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &last); err != nil {
			t.Fatalf("decode status: %v", err)
		}
		if last["running"] == false {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if last["running"] != false {
		t.Fatalf("ingest still running after timeout: %#v", last)
	}
	phase, _ := last["phase"].(string)
	if phase != "done" && phase != "error" {
		t.Fatalf("expected phase done|error got %q: %#v", phase, last)
	}
	logs, _ := last["logs"].([]any)
	if len(logs) == 0 {
		t.Fatalf("expected log lines: %#v", last)
	}
}
