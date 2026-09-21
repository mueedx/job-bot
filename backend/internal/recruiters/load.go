package recruiters

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const recruitersFileName = "recruiters.yaml"

// Load reads the user's recruiters.yaml if it exists, falling back to the
// shipped catalog. When filePath is empty, it is resolved relative to the
// data directory so the settings path resolution stays consistent.
func Load(dataDir string) []Recruiter {
	base := Catalog
	if dataDir == "" {
		dataDir = filepath.Join("..", "data")
	}
	path := filepath.Join(dataDir, recruitersFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return base
		}
		// A broken file is not fatal; the user gets the shipped catalog.
		return base
	}
	var catalog RecruiterCatalog
	if err := yaml.Unmarshal(b, &catalog); err != nil {
		// Malformed YAML is not fatal.
		return base
	}
	return Merge(base, catalog.Recruiters)
}

// Merge keeps every shipped entry and overlays user entries on top. A user
// entry with the same ID replaces the shipped one (override). A user entry
// with no ID is appended. Disabled entries (Website == "") are dropped.
// Order is stable: shipped entries first (by Catalog order), then user
// entries in file order.
func Merge(shipped, user []Recruiter) []Recruiter {
	kept := make([]Recruiter, 0, len(shipped))
	for _, r := range shipped {
		if r.Website == "" {
			continue
		}
		kept = append(kept, r)
	}
	userByID := map[string]int{}
	for i, r := range user {
		if r.ID != "" {
			userByID[r.ID] = i
		}
	}
	for id, i := range userByID {
		// Replace the shipped entry in place.
		for j := range kept {
			if kept[j].ID == id {
				kept[j] = user[i]
				break
			}
		}
	}
	// Append user entries whose ID doesn't match any kept (shipped or overridden)
	// entry, so a user-only recruiter (e.g. "c" above) gets added.
	keptIDs := map[string]bool{}
	for _, r := range kept {
		if r.ID != "" {
			keptIDs[r.ID] = true
		}
	}
	for _, r := range user {
		if r.ID == "" {
			if r.Website == "" {
				continue
			}
			kept = append(kept, r)
			continue
		}
		if !keptIDs[r.ID] {
			if r.Website == "" {
				continue
			}
			kept = append(kept, r)
		}
	}
	return Normalize(kept)
}

// Normalize trims whitespace and lowercases country/field tokens so lookups
// are case-insensitive and duplicates collapse.
func Normalize(rs []Recruiter) []Recruiter {
	out := make([]Recruiter, 0, len(rs))
	for _, r := range rs {
		r.ID = strings.TrimSpace(r.ID)
		r.Name = strings.TrimSpace(r.Name)
		r.Website = strings.TrimSpace(r.Website)
		countries := make([]string, 0, len(r.Countries))
		for _, c := range r.Countries {
			c = strings.ToLower(strings.TrimSpace(c))
			if c == "" {
				continue
			}
			countries = append(countries, c)
		}
		r.Countries = countries
		fields := make([]string, 0, len(r.Fields))
		seen := map[string]bool{}
		for _, f := range r.Fields {
			f = strings.ToLower(strings.TrimSpace(f))
			if f == "" || seen[f] {
				continue
			}
			seen[f] = true
			fields = append(fields, f)
		}
		r.Fields = fields
		out = append(out, r)
	}
	return out
}

// FilterByCountries returns only recruiters that hire in at least one enabled
// country. A recruiter with no Countries is treated as global and always
// returned. This is what the dashboard uses to hide recruiters for regions the
// user switched off.
func FilterByCountries(rs []Recruiter, enabled []string) []Recruiter {
	if len(enabled) == 0 {
		return rs
	}
	enabledSet := map[string]bool{}
	for _, c := range enabled {
		enabledSet[strings.ToLower(strings.TrimSpace(c))] = true
	}
	var out []Recruiter
	for _, r := range rs {
		if len(r.Countries) == 0 || enabledSet[strings.ToLower(r.Countries[0])] {
			out = append(out, r)
			continue
		}
		for _, c := range r.Countries {
			if enabledSet[c] {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// FilterByField returns only recruiters whose Fields overlap the given track
// tokens (case-insensitive).
func FilterByField(rs []Recruiter, trackTokens []string) []Recruiter {
	if len(trackTokens) == 0 {
		return rs
	}
	tokens := map[string]bool{}
	for _, t := range trackTokens {
		tokens[strings.ToLower(strings.TrimSpace(t))] = true
	}
	var out []Recruiter
	for _, r := range rs {
		for _, f := range r.Fields {
			if tokens[strings.ToLower(f)] {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// GroupByCountry returns recruiters keyed by country, with "global" for
// recruiters that list no country.
func GroupByCountry(rs []Recruiter) map[string][]Recruiter {
	out := map[string][]Recruiter{}
	for _, r := range rs {
		key := "global"
		if len(r.Countries) > 0 {
			key = strings.Join(r.Countries, ",")
		}
		out[key] = append(out[key], r)
	}
	return out
}

// WebsiteFor returns the website URL for a recruiter ID, or "" when not found.
func WebsiteFor(rs []Recruiter, id string) string {
	for _, r := range rs {
		if strings.EqualFold(r.ID, id) {
			return r.Website
		}
	}
	return ""
}
