package api

import "net/http"

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := s.Store.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if stats.ByStatus == nil {
		stats.ByStatus = map[string]int{}
	}
	stats.DailyLimit = dailyLimitFromEnv()
	writeJSON(w, http.StatusOK, stats)
}
