package web

import (
	"net/http"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/workflow"
)

type captureRequest struct {
	ExpectedVersion uint64              `json:"expectedVersion"`
	Actor           string              `json:"actor"`
	Run             *domain.CaptureRun  `json:"run"`
	Runs            []domain.CaptureRun `json:"runs"`
}
type resolutionRequest struct {
	ExpectedVersion  uint64 `json:"expectedVersion"`
	Actor            string `json:"actor"`
	FindingID        string `json:"findingID"`
	Resolution       string `json:"resolution"`
	ReplacementRunID string `json:"replacementRunID"`
	Action           string `json:"action"`
	ReasonCode       string `json:"reasonCode"`
	Description      string `json:"description"`
}

func (s *Server) AddCaptureRunHandler(w http.ResponseWriter, r *http.Request) {
	var req captureRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if (req.Run == nil) == (len(req.Runs) == 0) {
		writeError(w, domain.Invalid("请求必须且只能提供 run 或 runs"))
		return
	}
	if len(req.Runs) > 100 {
		writeError(w, domain.Invalid("runs 数组最多包含 100 行"))
		return
	}
	cmd := workflow.CaptureCommand{Meta: commandMeta(r, req.ExpectedVersion, req.Actor), Runs: req.Runs}
	if req.Run != nil {
		cmd.Run = *req.Run
	}
	batch, err := s.workflow.AddCaptureRun(batchID(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) RunQualityHandler(w http.ResponseWriter, r *http.Request) {
	var req metaRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	batch, err := s.workflow.RunQuality(batchID(r), commandMeta(r, req.ExpectedVersion, req.Actor))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) ResolveFindingHandler(w http.ResponseWriter, r *http.Request) {
	var req resolutionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	cmd := workflow.ResolveFindingCommand{Meta: commandMeta(r, req.ExpectedVersion, req.Actor), FindingID: req.FindingID, Resolution: req.Resolution, ReplacementRunID: req.ReplacementRunID, Action: req.Action, ReasonCode: req.ReasonCode, Description: req.Description}
	batch, err := s.workflow.ResolveFinding(batchID(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
