package web

import "net/http"

func (s *Server) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "tape-preservation-gate"})
}
