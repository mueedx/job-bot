// Package resumes finds the PDF resume that belongs to a job "track".
//
// Resumes live in one directory that the user owns (default <repo>/resumes).
// The directory is discovered at runtime so the same code works whether the
// server runs from the repo root, from backend/ (go run), or from /app (Docker).
//
// File naming is forgiving: `fullstack.pdf` always wins, but a file such as
// `Jane_Doe_Fullstack.pdf` is recognised by its name tokens, and `default.pdf`
// (or a lone PDF) covers any track without a dedicated file.
package resumes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirEnv optionally overrides the resume directory.
const DirEnv = "RESUME_DIR"

// DefaultDir is the directory name used when nothing else matches.
const DefaultDir = "resumes"

// DefaultFile serves tracks that have no dedicated PDF.
const DefaultFile = "default.pdf"

// Tracks lists the supported tracks in tie-break order (most specific first).
var Tracks = []string{"fde", "blockchain", "fullstack"}

// trackTokens maps a track to the filename tokens that identify it.
// Entries containing a space are matched as whole phrases; single words are
// matched as whole tokens, so "ai" never matches a filename containing "email".
var trackTokens = map[string][]string{
	"fde": {
		"fde", "ai", "agentic", "mcp", "rag",
		"forward deployed", "ai engineer", "solutions architect", "customer engineer",
	},
	"blockchain": {
		"blockchain", "web3", "solidity", "evm", "crypto", "defi", "substrate",
		"smart contract",
	},
	"fullstack": {
		"fullstack", "web", "react", "nestjs", "nodejs", "typescript",
		"node js", "full stack",
	},
}

// Dir returns the directory that holds resume PDFs.
//
// Order of preference:
//  1. $RESUME_DIR (used exactly as given)
//  2. ./resumes  — repo root, or /app inside Docker
//  3. ../resumes — server started from backend/
//  4. ./resumes  — default; the user creates it
func Dir() string {
	if dir := strings.TrimSpace(os.Getenv(DirEnv)); dir != "" {
		return dir
	}
	if isDir(DefaultDir) {
		return DefaultDir
	}
	if parent := filepath.Join("..", DefaultDir); isDir(parent) {
		return parent
	}
	return DefaultDir
}

// Entry describes one resume PDF found on disk.
type Entry struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Track   string `json:"track"`
	Default bool   `json:"default"`
}

// Status summarises what the server can see on disk, for logs and the API.
type Status struct {
	Dir      string            `json:"dir"`
	Files    []string          `json:"files"`
	Tracks   map[string]string `json:"tracks"`
	Missing  []string          `json:"missing"`
	Fallback string            `json:"fallback,omitempty"`
}

// IsTrack reports whether name is a supported track.
func IsTrack(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, t := range Tracks {
		if t == name {
			return true
		}
	}
	return false
}

// Path returns the stored resume path for a track, for example
// "resumes/Jane_Doe_Fullstack.pdf".
//
// The value is always expressed as DefaultDir/<basename>, so records stay
// portable between Docker (/app/resumes) and local runs; use Resolve to get the
// real file path. An unknown track is an error. A known track with no matching
// file returns the conventional "resumes/<track>.pdf" placeholder (reported by
// Inspect as missing) so callers can still record the intent.
func Path(track string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(track))
	if !IsTrack(t) {
		return "", fmt.Errorf("invalid track %q (want one of: %s)", track, strings.Join(Tracks, ", "))
	}

	files := pdfFiles(Dir())

	for _, name := range files {
		if normalize(name) == t {
			return rel(name), nil
		}
	}

	best, bestScore := "", 0
	for _, name := range files {
		s := score(normalize(name), t)
		if s == 0 {
			continue
		}
		if s > bestScore || (s == bestScore && prefer(name, best)) {
			best, bestScore = name, s
		}
	}
	if best != "" {
		return rel(best), nil
	}
	if contains(files, DefaultFile) {
		return rel(DefaultFile), nil
	}
	// A lone PDF serves every track, but only when its name does not clearly
	// belong to a different track (a single web3.pdf must not stand in for
	// fullstack).
	if len(files) == 1 {
		if _, s := bestTrack(normalize(files[0])); s == 0 {
			return rel(files[0]), nil
		}
	}
	return rel(t + ".pdf"), nil
}

// Resolve turns a stored resume path into the real file path on disk.
func Resolve(stored string) string {
	return filepath.Join(Dir(), filepath.Base(stored))
}

// Exists reports whether the file behind a stored resume path is present.
func Exists(stored string) bool {
	_, err := os.Stat(Resolve(stored))
	return err == nil
}

// Track extracts the track from a stored resume path. Unknown names fall back
// to "fullstack", mirroring the matcher's default.
func Track(path string) string {
	if path == "" {
		return "fullstack"
	}
	if t, s := bestTrack(normalize(filepath.Base(path))); s > 0 {
		return t
	}
	return "fullstack"
}

// Available lists the resume PDFs currently on disk, with the track each one
// appears to serve.
func Available() []Entry {
	dir := Dir()
	files := pdfFiles(dir)
	entries := make([]Entry, 0, len(files))
	for _, name := range files {
		entries = append(entries, Entry{
			File:    name,
			Path:    rel(name),
			Track:   Track(name),
			Default: normalize(name) == normalize(DefaultFile),
		})
	}
	return entries
}

// Inspect reports the resolved directory and which tracks have a real file.
func Inspect() Status {
	dir := Dir()
	files := pdfFiles(dir)

	st := Status{
		Dir:    dir,
		Files:  files,
		Tracks: map[string]string{},
	}
	if contains(files, DefaultFile) {
		st.Fallback = rel(DefaultFile)
	}
	for _, t := range Tracks {
		p, err := Path(t)
		if err != nil {
			continue
		}
		st.Tracks[t] = p
		if !Exists(p) {
			st.Missing = append(st.Missing, t)
		}
	}
	if st.Files == nil {
		st.Files = []string{}
	}
	if st.Missing == nil {
		st.Missing = []string{}
	}
	sort.Strings(st.Missing)
	return st
}

// rel renders a stored resume path for a file name.
func rel(name string) string {
	return DefaultDir + "/" + name
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// pdfFiles returns sorted "*.pdf" base names in dir, ignoring hidden files.
func pdfFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".pdf") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

// normalize lowercases a file name and reduces every run of non-alphanumeric
// characters to a single space: "Jane_Doe_AI-Engineer.pdf" -> "jane doe ai engineer".
func normalize(name string) string {
	name = strings.TrimSuffix(strings.ToLower(filepath.Base(name)), ".pdf")

	var b strings.Builder
	space := true // suppress leading separators
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			space = false
			continue
		}
		if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

// score counts how strongly a normalized file name signals a track.
// Phrases count double so "forward deployed" beats a stray "ai" token.
func score(normalized, track string) int {
	total := 0
	for _, kw := range trackTokens[track] {
		if containsPhrase(normalized, kw) {
			if strings.Contains(kw, " ") {
				total += 2
			} else {
				total++
			}
		}
	}
	return total
}

// bestTrack returns the track with the strongest signal in a normalized file
// name, plus that score (0 when the name signals no track at all).
func bestTrack(normalized string) (string, int) {
	best, bestScore := "", 0
	for _, t := range Tracks {
		if s := score(normalized, t); s > bestScore {
			best, bestScore = t, s
		}
	}
	return best, bestScore
}

func containsPhrase(normalized, keyword string) bool {
	return strings.Contains(" "+normalized+" ", " "+keyword+" ")
}

// prefer breaks ties deterministically: shortest name, then alphabetical.
func prefer(candidate, current string) bool {
	if current == "" {
		return true
	}
	if len(candidate) != len(current) {
		return len(candidate) < len(current)
	}
	return candidate < current
}
