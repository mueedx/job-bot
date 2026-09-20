package recruiters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogIdempotentAcrossLoads(t *testing.T) {
	// Loading the recruiter catalog twice must produce identical results so the
	// /api/recruiters response is stable across requests.
	r1 := Load("")
	r2 := Load("")
	if len(r1) != len(r2) {
		t.Fatalf("Load() returned %d recruiters first time and %d second time", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].ID != r2[i].ID || r1[i].Name != r2[i].Name || r1[i].Website != r2[i].Website {
			t.Fatalf("recruiter[%d] differs: %+v vs %+v", i, r1[i], r2[i])
		}
	}
}

func TestCatalogContainsVerifiedRecruiters(t *testing.T) {
	// The shipped catalog must contain at least one verified entry, otherwise no
	// recruiter UI is usable without the user maintaining their own file.
	found := 0
	for _, r := range Catalog {
		if r.Verified {
			found++
			if r.Website == "" || r.Name == "" {
				t.Errorf("verified recruiter %q has empty name or website", r.ID)
			}
		}
	}
	if found == 0 {
		t.Fatal("catalog has no verified recruiters")
	}
}

func TestMergeOverride(t *testing.T) {
	shipped := []Recruiter{
		{ID: "a", Name: "Alpha", Website: "https://alpha.example", Countries: []string{"de"}, Fields: []string{"fullstack"}},
		{ID: "b", Name: "Bravo", Website: "https://bravo.example", Countries: []string{"gb"}, Fields: []string{"blockchain"}},
	}
	user := []Recruiter{
		{ID: "a", Name: "Alpha (updated)", Website: "https://alpha.v2", Countries: []string{"de", "at"}, Fields: []string{"fullstack", "devops"}},
		{ID: "c", Name: "Charlie", Website: "https://charlie.example", Countries: []string{"ie"}, Fields: []string{"fde"}},
	}
	got := Merge(shipped, user)
	if len(got) != 3 {
		t.Fatalf("Merge() = %d entries, want 3", len(got))
	}
	byID := map[string]Recruiter{}
	for _, r := range got {
		byID[r.ID] = r
	}
	if got, want := byID["a"].Name, "Alpha (updated)"; got != want {
		t.Errorf("overridden entry a.Name = %q, want %q", got, want)
	}
	if got, want := byID["a"].Countries, []string{"de", "at"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("overridden entry a.Countries = %v, want %v", got, want)
	}
	if _, ok := byID["b"]; !ok {
		t.Error("entry b should have been preserved")
	}
	if byID["c"].Name != "Charlie" {
		t.Errorf("appended entry c.Name = %q", byID["c"].Name)
	}
}

func TestMergeDropsDisabled(t *testing.T) {
	shipped := []Recruiter{
		{ID: "a", Name: "Alpha", Website: "https://a.example", Countries: []string{"de"}},
		{ID: "b", Name: "Bravo (disabled)", Website: "", Countries: []string{"gb"}},
	}
	got := Merge(shipped, nil)
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("disabled shipped entry should be dropped: %v", got)
	}
}

func TestFilterByCountries(t *testing.T) {
	rs := []Recruiter{
		{ID: "a", Name: "A", Website: "https://a.example", Countries: []string{"de"}},
		{ID: "b", Name: "B", Website: "https://b.example", Countries: []string{"gb"}},
		{ID: "c", Name: "C", Website: "https://c.example", Countries: []string{"de", "at"}},
		{ID: "d", Name: "D (global)", Website: "https://d.example", Countries: nil},
	}
	// Filter to de and at: should include a, c, d.
	got := FilterByCountries(rs, []string{"de", "at"})
	if len(got) != 3 {
		t.Fatalf("FilterByCountries(de,at) = %d, want 3", len(got))
	}
	ids := map[string]bool{}
	for _, r := range got {
		ids[r.ID] = true
	}
	if !ids["a"] || !ids["c"] || !ids["d"] || ids["b"] {
		t.Errorf("FilterByCountries(de,at) = %v, want a,c,d", ids)
	}
}

func TestFilterByField(t *testing.T) {
	rs := []Recruiter{
		{ID: "a", Name: "A", Website: "https://a.example", Fields: []string{"fullstack", "backend"}},
		{ID: "b", Name: "B", Website: "https://b.example", Fields: []string{"blockchain"}},
		{ID: "c", Name: "C", Website: "https://c.example", Fields: []string{"frontend", "react"}},
	}
	got := FilterByField(rs, []string{"fullstack"})
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("FilterByField(fullstack) = %v, want [a]", got)
	}
	got = FilterByField(rs, []string{"react"})
	if len(got) != 1 || got[0].ID != "c" {
		t.Fatalf("FilterByField(react) = %v, want [c]", got)
	}
}

func TestWebsiteFor(t *testing.T) {
	rs := []Recruiter{
		{ID: "alpha", Name: "Alpha", Website: "https://alpha.example"},
		{ID: "beta", Name: "Beta", Website: "https://beta.example"},
	}
	if got := WebsiteFor(rs, "alpha"); got != "https://alpha.example" {
		t.Errorf("WebsiteFor(alpha) = %q", got)
	}
	if got := WebsiteFor(rs, "missing"); got != "" {
		t.Errorf("WebsiteFor(missing) = %q, want empty", got)
	}
}

func TestLoadUserFile(t *testing.T) {
	// Load() should read and merge a user recruiters.yaml when present.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "recruiters.yaml"), []byte(`
recruiters:
  - id: user-added
    name: "User Added Recruiter"
    website: "https://user.example"
    countries: [ie, gb]
    fields: [fullstack, blockchain]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	found := false
	for _, r := range got {
		if r.ID == "user-added" {
			found = true
			if r.Countries[0] != "ie" || r.Countries[1] != "gb" {
				t.Errorf("user entry countries = %v, want [ie, gb]", r.Countries)
			}
		}
	}
	if !found {
		t.Fatal("user recruiters.yaml was not loaded")
	}
}

func TestLoadMalformedFileFallsBackToCatalog(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "recruiters.yaml"), []byte("recruiters:\n  - {bad: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if len(got) != len(Catalog) {
		t.Fatalf("malformed recruiters.yaml should fall back to shipped catalog: got %d, want %d", len(got), len(Catalog))
	}
}
