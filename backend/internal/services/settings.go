package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/scrapers"
	"gopkg.in/yaml.v3"
)

// DefaultCountries is the region target when the user has configured nothing:
// the explicitly useful English-speaking markets. Lowercase ISO-3166 alpha-2.
var DefaultCountries = []string{"ie", "gb", "pk", "ae", "sa", "au", "nz", "us", "ca"}

// Settings is the user-editable runtime configuration: which job sources run,
// and which regions to target. Stored as data/settings.yaml so it can be
// edited by hand or from the dashboard, and survives restarts.
type Settings struct {
	Sources map[string]bool `json:"sources" yaml:"sources"`
	// RecruiterCountries are the target regions (lowercase ISO-3166 alpha-2).
	// They scope country-aware sources and the recruiter directory.
	RecruiterCountries []string `json:"recruiter_countries" yaml:"recruiter_countries"`
}

// SettingsPath is the settings file name inside the data directory.
const SettingsPath = "settings.yaml"

// DefaultSettings returns the settings used when no file exists yet: every
// known source on, and the default region list.
func DefaultSettings() *Settings {
	sources := make(map[string]bool, len(scrapers.All()))
	for _, spec := range scrapers.All() {
		sources[spec.Name] = true
	}
	countries := append([]string(nil), DefaultCountries...)
	return &Settings{Sources: sources, RecruiterCountries: countries}
}

func settingsFile(dataDir string) string {
	if dataDir == "" {
		return filepath.Join("data", SettingsPath)
	}
	return filepath.Join(dataDir, SettingsPath)
}

// LoadSettings reads settings.yaml, falling back to defaults when the file does
// not exist (the expected first-run state) and repairing partial files so a
// hand-edited file cannot take every source offline by omission.
func LoadSettings(dataDir string) (*Settings, error) {
	settings := DefaultSettings()
	path := settingsFile(dataDir)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var parsed Settings
	if err := yaml.Unmarshal(b, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for name, on := range parsed.Sources {
		settings.Sources[name] = on
	}
	if len(parsed.RecruiterCountries) > 0 {
		settings.RecruiterCountries = parsed.RecruiterCountries
	}
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
	}
	return settings, nil
}

// Validate normalises country codes and rejects unknown source names, so a
// typo in settings.yaml is reported instead of silently doing nothing.
func (s *Settings) Validate() error {
	if s.Sources == nil {
		s.Sources = map[string]bool{}
	}
	for name := range s.Sources {
		if _, ok := scrapers.Lookup(name); !ok {
			return fmt.Errorf("unknown source %q", name)
		}
	}
	codes := make([]string, 0, len(s.RecruiterCountries))
	seen := map[string]bool{}
	for _, c := range s.RecruiterCountries {
		c = strings.ToLower(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		if len(c) != 2 || !isLowerAlpha(c) {
			return fmt.Errorf("country code %q is not a 2-letter code (e.g. ie, gb, ae)", c)
		}
		if !seen[c] {
			seen[c] = true
			codes = append(codes, c)
		}
	}
	s.RecruiterCountries = codes
	return nil
}

func isLowerAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// SaveSettings writes settings.yaml atomically (temp file + rename) so a crash
// mid-write cannot leave a truncated config behind.
func SaveSettings(dataDir string, settings *Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	path := settingsFile(dataDir)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	b, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	return os.Rename(tmp, path)
}
