package api

import (
	"context"
	"net/http"
	"time"

	"github.com/mueedx/job-bot/backend/internal/resumes"
)

// handleListResumes reports where resume PDFs are read from, which file each
// track resolves to, and what is still missing. It lets users confirm
// detection from the API instead of digging through server logs.
func (s *Server) handleListResumes(w http.ResponseWriter, _ *http.Request) {
	st := resumes.Inspect()
	entries := resumes.Available()
	profiles := s.listProfiles()
	writeJSON(w, http.StatusOK, map[string]any{
		"dir":      st.Dir,
		"files":    st.Files,
		"tracks":   st.Tracks,
		"missing":  st.Missing,
		"fallback": st.Fallback,
		"entries":  entries,
		"profiles": profiles,
	})
}

// resumeProfileView is the JSON shape returned to the frontend.
type resumeProfileView struct {
	ResumePath  string `json:"resume_path"`
	ContentHash string `json:"content_hash"`
	Track       string `json:"track"`
	Skills      string `json:"skills"`
	Keywords    string `json:"keywords"`
	Seniority   string `json:"seniority"`
	Summary     string `json:"summary"`
	Source      string `json:"source"`
	ExtractedAt string `json:"extracted_at"`
}

func (s *Server) listProfiles() []resumeProfileView {
	if s.Store == nil {
		return nil
	}
	rows, _ := s.Store.ListResumeProfiles()
	out := make([]resumeProfileView, 0, len(rows))
	for _, p := range rows {
		out = append(out, resumeProfileView{
			ResumePath:  p.ResumePath,
			ContentHash: p.ContentHash,
			Track:       p.Track,
			Skills:      derefStr(p.Skills),
			Keywords:    derefStr(p.Keywords),
			Seniority:   p.Seniority,
			Summary:     p.Summary,
			ExtractedAt: p.ExtractedAt.Format(time.RFC3339),
		})
	}
	return out
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// handleAnalyzeResume runs AI/resume-text extraction on every resume PDF and
// returns the extracted profiles. Without OPENAI_API_KEY this still runs but
// only fills in filename-based track detection (the keyless fallback).
func (s *Server) handleAnalyzeResume(w http.ResponseWriter, _ *http.Request) {
	if s.Analyzer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "analyzer not configured",
		})
		return
	}
	entries := resumes.Available()
	ctx := context.Background()
	type result struct {
		Path   string `json:"path"`
		Error  string `json:"error,omitempty"`
		Source string `json:"source,omitempty"`
		Track  string `json:"track,omitempty"`
	}
	var results []result
	for _, e := range entries {
		p, err := s.Analyzer.AnalyzePDF(ctx, e.Path)
		if err != nil {
			results = append(results, result{Path: e.Path, Error: err.Error()})
			continue
		}
		results = append(results, result{Path: e.Path, Source: p.Source, Track: p.Track})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"analyzed": len(results),
		"results":  results,
	})
}
