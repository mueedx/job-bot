package api

import (
	"net/http"

	"github.com/mueedx/job-bot/backend/internal/resumes"
)

// handleListResumes reports where resume PDFs are read from, which file each
// track resolves to, and what is still missing. It lets users confirm
// detection from the API instead of digging through server logs.
func (s *Server) handleListResumes(w http.ResponseWriter, _ *http.Request) {
	st := resumes.Inspect()
	writeJSON(w, http.StatusOK, map[string]any{
		"dir":      st.Dir,
		"files":    st.Files,
		"tracks":   st.Tracks,
		"missing":  st.Missing,
		"fallback": st.Fallback,
		"entries":  resumes.Available(),
	})
}
