package web

import (
	"net/http"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/workflow"
)

type createBatchRequest struct {
	ID            string               `json:"id"`
	Title         string               `json:"title"`
	Operator      string               `json:"operator"`
	Reviewer      string               `json:"reviewer"`
	Actor         string               `json:"actor"`
	TargetProfile domain.TargetProfile `json:"targetProfile"`
}

func (s *Server) ListBatchesHandler(w http.ResponseWriter, _ *http.Request) {
	items, err := s.workflow.ListBatches()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) CreateBatchHandler(w http.ResponseWriter, r *http.Request) {
	var req createBatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	cmd := workflow.CreateBatchCommand{ID: req.ID, Title: req.Title, Operator: req.Operator, Reviewer: req.Reviewer, TargetProfile: req.TargetProfile, Meta: commandMeta(r, 0, req.Actor)}
	batch, err := s.workflow.CreateBatch(cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, batch)
}

func (s *Server) GetBatchHandler(w http.ResponseWriter, r *http.Request) {
	view, err := s.workflow.BatchView(batchID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}
