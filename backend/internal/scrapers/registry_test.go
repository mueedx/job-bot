package scrapers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestJobicyFetch(t *testing.T) {
	raw, err := os.ReadFile("testdata/jobicy.json")
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/remote-jobs" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("missing Accept: application/json header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Timeout = 5 * time.Second
	j := &Jobicy{Client: client, BaseURL: ts.URL}
	jobs, err := j.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got, want := len(jobs), 3; got != want {
		t.Fatalf("got %d jobs, want %d", got, want)
	}

	// First job: USA, remote false (not "anywhere")
	got := jobs[0]
	if got.Source != "jobicy" || got.SourceID != "151146" {
		t.Errorf("job[0] id = %+v", got)
	}
	if got.Title != "AWS Data Engineer (Senior)" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Company != "Mactores" {
		t.Errorf("company = %q", got.Company)
	}
	if got.Location != "USA" {
		t.Errorf("location = %q", got.Location)
	}
	if got.IsRemote {
		t.Error("USA job should not be remote")
	}
	if got.URL != "https://jobicy.com/jobs/151146-aws-data-engineer-senior" {
		t.Errorf("url = %q", got.URL)
	}

	// Third job: Anywhere, remote true
	got = jobs[2]
	if got.Title != "Blockchain Developer" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Company != "ChainLabs" {
		t.Errorf("company = %q", got.Company)
	}
	if got.Location != "Anywhere" {
		t.Errorf("location = %q", got.Location)
	}
	if !got.IsRemote {
		t.Error("Anywhere job should be remote")
	}

	// All jobs must have non-empty descriptions (HTMLToText ran).
	for _, jb := range jobs {
		if jb.Description == "" {
			t.Error("job has empty description after HTMLToText")
		}
	}
}

func TestJobicyGeoFilter(t *testing.T) {
	raw, err := os.ReadFile("testdata/jobicy.json")
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the geo param is forwarded.
		if r.URL.Query().Get("geo") != "uk" {
			t.Errorf("expected geo=uk, got %s", r.URL.Query().Get("geo"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer ts.Close()

	j := &Jobicy{Client: ts.Client(), BaseURL: ts.URL, Geo: "uk"}
	_, err = j.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestJobicyEmptyOrMissingFields(t *testing.T) {
	// A row with an empty title or company must be skipped.
	raw := []byte(`{
		"success": true,
		"jobs": [
			{"id":1, "url":"https://x.com/1", "jobTitle":"", "companyName":"Co", "jobGeo":"UK", "jobDescription":"<p>x</p>", "pubDate":"2026-01-01T00:00:00+00:00"},
			{"id":2, "url":"https://x.com/2", "jobTitle":"Real", "companyName":"", "jobGeo":"UK", "jobDescription":"<p>x</p>", "pubDate":"2026-01-01T00:00:00+00:00"},
			{"id":3, "url":"https://x.com/3", "jobTitle":"OK", "companyName":"Co", "jobGeo":"UK", "jobDescription":"<p>x</p>", "pubDate":"2026-01-01T00:00:00+00:00"}
		]
	}`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer ts.Close()

	j := &Jobicy{Client: ts.Client(), BaseURL: ts.URL}
	jobs, err := j.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got, want := len(jobs), 1; got != want {
		t.Fatalf("got %d jobs after skipping empty rows, want %d", got, want)
	}
	if jobs[0].Title != "OK" {
		t.Errorf("expected the only valid job, got %q", jobs[0].Title)
	}
}

func TestArbeitnowFetch(t *testing.T) {
	raw, err := os.ReadFile("testdata/arbeitnow.json")
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/job-board-api" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer ts.Close()

	a := &Arbeitnow{Client: ts.Client(), BaseURL: ts.URL}
	jobs, err := a.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got, want := len(jobs), 3; got != want {
		t.Fatalf("got %d jobs, want %d", got, want)
	}

	got := jobs[0]
	if got.Source != "arbeitnow" || got.SourceID != "aws-data-engineer-senior-12345" {
		t.Errorf("job[0] = %+v", got)
	}
	if got.Title != "AWS Data Engineer (Senior)" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Company != "Mactores" {
		t.Errorf("company = %q", got.Company)
	}
	if got.Location != "Remote" {
		t.Errorf("location = %q", got.Location)
	}
	if !got.IsRemote {
		t.Error("remote=true job should be flagged remote")
	}

	got = jobs[1]
	if got.Location != "London" {
		t.Errorf("location = %q", got.Location)
	}
	if got.IsRemote {
		t.Error("London on-site job should not be remote")
	}

	// Third job: remote=true flag
	got = jobs[2]
	if got.Location != "Anywhere" {
		t.Errorf("location = %q", got.Location)
	}
	if !got.IsRemote {
		t.Error("Anywhere job should be remote")
	}

	for _, jb := range jobs {
		if jb.Description == "" {
			t.Error("job has empty description after HTMLToText")
		}
	}
}

func TestInfoShapeIsStable(t *testing.T) {
	info := Info(nil)
	if len(info) != len(Registry) {
		t.Fatalf("Info() returned %d entries, want %d", len(info), len(Registry))
	}
	for i := 1; i < len(info); i++ {
		if info[i-1].Label > info[i].Label {
			t.Fatalf("Info() is not sorted by label: %q before %q", info[i-1].Label, info[i].Label)
		}
	}
	for _, s := range info {
		// The dashboard maps over these; nil would serialise as null and crash it.
		if s.Countries == nil || s.EnvKeys == nil {
			t.Errorf("%s has nil slices (countries/keys)", s.Name)
		}
		if s.Name == "" || s.Label == "" {
			t.Errorf("incomplete source info: %+v", s)
		}
		if s.Ready != (s.Reason == "") {
			t.Errorf("%s: Ready=%v but Reason=%q", s.Name, s.Ready, s.Reason)
		}
	}
}
