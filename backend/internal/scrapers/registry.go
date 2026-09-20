package scrapers

import (
	"net/http"
	"os"
	"sort"
	"strings"
)

// Kind classifies a source so the dashboard can group them and explain why one
// is switched off.
type Kind string

const (
	// KindBoard is a set of company ATS boards (Greenhouse, Lever, Ashby…).
	KindBoard Kind = "board"
	// KindAggregator is a third-party job search API covering many companies.
	KindAggregator Kind = "aggregator"
	// KindFeed is a simple public job feed.
	KindFeed Kind = "feed"
	// KindGated is opt-in only: the site restricts automated access, so the
	// source stays off unless the user enables it explicitly, and may still fail.
	KindGated Kind = "gated"
)

// Deps is what a source needs to be constructed. It is built once per ingest run
// so every source sees the same HTTP client and curated configuration.
type Deps struct {
	Client    *http.Client
	Targets   *TargetCompanies
	Countries []string // enabled target country codes (lowercase ISO-3166 alpha-2)
}

// Spec is the single source of truth for a job source: the metadata the API and
// dashboard render, plus the constructor used by the ingestor. Adding a source
// means adding one entry to Registry — nothing else needs to change.
//
// Name must stay equal to the value written into jobs.source: the dashboard,
// the paywall/attribution notes and existing database rows all key off it.
type Spec struct {
	Name  string // stable id, matches jobs.source
	Label string // human-readable name for the UI

	Kind Kind
	// Countries are the country codes this source can target. Empty means the
	// source is global and ignores country settings.
	Countries []string
	// EnvKeys are environment variables that must be set before the source can
	// run (bring-your-own API key sources). Empty means it works out of the box.
	EnvKeys []string
	// Note is the paywall or attribution obligation shown next to the source.
	Note string
	// OptInEnv is an env var that must be exactly "true" for the source to run.
	// Used by gated sources that are off by default for legal/ToS reasons.
	OptInEnv string
	// Fragile marks a source known to break or get blocked by the site.
	Fragile bool
	// Unverified marks a source whose parsing has never been confirmed against
	// live markup. The health check reports it as unverified, never as healthy.
	Unverified bool
	// NeedsTargets marks a source that reads curated company slugs from
	// data/target_companies.yaml. Without them it would otherwise "succeed" while
	// returning nothing, which is the silent failure the health check exists for.
	NeedsTargets bool

	Build func(Deps) Scraper
}

// targetSlugs returns the curated board slugs for a source, tolerating a nil
// Targets so a misconfigured run degrades to "no slugs" rather than panicking.
func targetSlugs(d Deps, source string) []string {
	if d.Targets == nil {
		return nil
	}
	switch source {
	case "greenhouse":
		return d.Targets.Greenhouse
	case "lever":
		return d.Targets.Lever
	case "ashby":
		return d.Targets.Ashby
	default:
		return nil
	}
}

// Slugs returns the curated board slugs this source would use, so callers can
// tell "no postings found" apart from "nothing was configured to look at".
func (s Spec) Slugs(t *TargetCompanies) []string {
	return targetSlugs(Deps{Targets: t}, s.Name)
}

// All returns a copy of the registry.
func All() []Spec {
	return append([]Spec(nil), Registry...)
}

// Lookup finds a source by name.
func Lookup(name string) (Spec, bool) {
	for _, s := range Registry {
		if s.Name == name {
			return s, true
		}
	}
	return Spec{}, false
}

// Label returns the human-readable name for a source, falling back to the raw
// name so an unknown (e.g. legacy) source still renders sensibly.
func Label(name string) string {
	if s, ok := Lookup(name); ok && s.Label != "" {
		return s.Label
	}
	return name
}

// Ready reports whether the source can run right now, plus a human-readable
// reason when it cannot (missing key, opt-in not enabled).
func (s Spec) Ready() (bool, string) {
	for _, key := range s.EnvKeys {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return false, "needs " + key
		}
	}
	if s.OptInEnv != "" && !strings.EqualFold(strings.TrimSpace(os.Getenv(s.OptInEnv)), "true") {
		return false, "disabled by default — set " + s.OptInEnv + "=true to enable"
	}
	return true, ""
}

// SupportsCountry reports whether a source targets the given country code.
// A source with no Countries is global and supports everything.
func (s Spec) SupportsCountry(code string) bool {
	if len(s.Countries) == 0 {
		return true
	}
	code = strings.ToLower(strings.TrimSpace(code))
	for _, c := range s.Countries {
		if c == code {
			return true
		}
	}
	return false
}

// Enabled returns the specs that should run: those that are Ready and have not
// been switched off in settings. A nil settings map means "use each source's own
// default", which is what the app does before any settings file exists.
func Enabled(settings map[string]bool) []Spec {
	var out []Spec
	for _, s := range Registry {
		if ok, _ := s.Ready(); !ok {
			continue
		}
		if settings != nil {
			if on, set := settings[s.Name]; set && !on {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

// BuildAll constructs the scrapers for the enabled sources.
func BuildAll(d Deps, settings map[string]bool) []Scraper {
	specs := Enabled(settings)
	out := make([]Scraper, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Build(d))
	}
	return out
}

// SkippedSource records a source that was left out of a run and why, so the
// ingest log can say "not disabled, just out of scope".
type SkippedSource struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

// BuildScoped constructs the enabled scrapers, additionally skipping
// country-scoped sources whose supported countries are all switched off.
// It returns the scrapers plus every skip decision for logging.
func BuildScoped(d Deps, settings map[string]bool, countries []string) ([]Scraper, []SkippedSource) {
	var out []Scraper
	var skipped []SkippedSource
	for _, s := range Enabled(settings) {
		if reason := s.ScopeReason(countries); reason != "" {
			skipped = append(skipped, SkippedSource{Name: s.Name, Label: s.Label, Reason: reason})
			continue
		}
		out = append(out, s.Build(d))
	}
	return out, skipped
}

// ScopeReason is empty when the source should run for the enabled countries, or
// a human-readable reason when every country it supports is switched off.
// Global sources (no Countries) always run.
func (s Spec) ScopeReason(enabledCountries []string) string {
	if len(s.Countries) == 0 || len(enabledCountries) == 0 {
		return ""
	}
	for _, c := range enabledCountries {
		if s.SupportsCountry(c) {
			return ""
		}
	}
	return "supports none of the enabled regions (supports: " + strings.Join(s.Countries, ", ") + ")"
}

// SourceInfo is the API shape for one source: its metadata plus whether it can
// run right now, and why not when it cannot.
type SourceInfo struct {
	Name         string   `json:"name"`
	Label        string   `json:"label"`
	Kind         Kind     `json:"kind"`
	Countries    []string `json:"countries"`
	EnvKeys      []string `json:"env_keys"`
	Note         string   `json:"note,omitempty"`
	OptInEnv     string   `json:"opt_in_env,omitempty"`
	Fragile      bool     `json:"fragile"`
	Unverified   bool     `json:"unverified"`
	NeedsTargets bool     `json:"needs_targets"`
	Ready        bool     `json:"ready"`
	Reason       string   `json:"reason,omitempty"`
	Enabled      bool     `json:"enabled"`
}

// Info describes every known source for the dashboard, sorted by label so the UI
// order is stable. settings == nil means "use each source's own default".
func Info(settings map[string]bool) []SourceInfo {
	out := make([]SourceInfo, 0, len(Registry))
	for _, s := range Registry {
		ready, reason := s.Ready()
		enabled := ready
		if settings != nil {
			if on, set := settings[s.Name]; set {
				enabled = ready && on
			}
		}
		countries := s.Countries
		if countries == nil {
			countries = []string{}
		}
		keys := s.EnvKeys
		if keys == nil {
			keys = []string{}
		}
		out = append(out, SourceInfo{
			Name:         s.Name,
			Label:        s.Label,
			Kind:         s.Kind,
			Countries:    countries,
			EnvKeys:      keys,
			Note:         s.Note,
			OptInEnv:     s.OptInEnv,
			Fragile:      s.Fragile,
			Unverified:   s.Unverified,
			NeedsTargets: s.NeedsTargets,
			Ready:        ready,
			Reason:       reason,
			Enabled:      enabled,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// Registry is every source the app knows about. Only sources that pass Ready
// (and are not switched off in settings) actually run during an ingest.
var Registry = []Spec{
	{
		Name:         "greenhouse",
		Label:        "Greenhouse",
		Kind:         KindBoard,
		NeedsTargets: true,
		Build: func(d Deps) Scraper {
			return &Greenhouse{Client: d.Client, Slugs: targetSlugs(d, "greenhouse")}
		},
	},
	{
		Name:         "lever",
		Label:        "Lever",
		Kind:         KindBoard,
		NeedsTargets: true,
		Build: func(d Deps) Scraper {
			return &Lever{Client: d.Client, Slugs: targetSlugs(d, "lever")}
		},
	},
	{
		Name:         "ashby",
		Label:        "Ashby",
		Kind:         KindBoard,
		NeedsTargets: true,
		Build: func(d Deps) Scraper {
			return &Ashby{Client: d.Client, Slugs: targetSlugs(d, "ashby")}
		},
	},
	{
		Name:  "remoteok",
		Label: "RemoteOK",
		Kind:  KindFeed,
		Note:  "Applying may require a RemoteOK Premium subscription or login",
		Build: func(d Deps) Scraper { return &RemoteOK{Client: d.Client} },
	},
	{
		// The crypto scraper reads RemoteOK's tag feed, so it inherits the same
		// premium gate. Name is "web3" (not "cryptojobs") because that is the
		// value older rows already store in jobs.source.
		Name:  "web3",
		Label: "Crypto / Web3",
		Kind:  KindFeed,
		Note:  "Mirrors RemoteOK listings — applying may require RemoteOK Premium or a login",
		Build: func(d Deps) Scraper { return &CryptoJobs{Client: d.Client} },
	},
	{
		Name:  "jobicy",
		Label: "Jobicy",
		Kind:  KindAggregator,
		// Jobicy has no dedicated country filter; the geo slug covers broad regions
		// (anywhere, apac, emea, latam, and individual countries). Leaving Countries
		// empty means it runs for every target list and the UI shows its broad reach.
		Note:  "If you build on this feed, credit Jobicy with a link to jobicy.com",
		Build: func(d Deps) Scraper { return &Jobicy{Client: d.Client} },
	},
	{
		Name:      "arbeitnow",
		Label:     "Arbeitnow",
		Kind:      KindAggregator,
		Countries: []string{"de"},
		Note:      "Mostly German/European listings; enable only if your target regions include Germany or nearby",
		Build:     func(d Deps) Scraper { return &Arbeitnow{Client: d.Client} },
	},
}
