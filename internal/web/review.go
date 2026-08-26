package web

import (
	"net/http"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/workflow"
)

type reviewRequest struct {
	ExpectedVersion uint64                `json:"expectedVersion"`
	Actor           string                `json:"actor"`
	Decision        domain.Decision       `json:"decision"`
	ReasonCodes     []string              `json:"reasonCodes"`
	Reasons         []domain.ReviewReason `json:"reasons"`
	Comment         string                `json:"comment"`
}

type reviewSubmissionRequest struct {
	ExpectedVersion uint64                         `json:"expectedVersion"`
	Actor           string                         `json:"actor"`
	Remediations    []domain.RemediationResolution `json:"remediations"`
}

func (s *Server) SubmitReviewHandler(w http.ResponseWriter, r *http.Request) {
	var req reviewSubmissionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	batch, err := s.workflow.SubmitReviewWithRemediations(batchID(r), workflow.SubmitReviewCommand{Meta: commandMeta(r, req.ExpectedVersion, req.Actor), Remediations: req.Remediations})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) DecideReviewHandler(w http.ResponseWriter, r *http.Request) {
	var req reviewRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	cmd := workflow.ReviewCommand{Meta: commandMeta(r, req.ExpectedVersion, req.Actor), Decision: req.Decision, ReasonCodes: req.ReasonCodes, Reasons: req.Reasons, Comment: req.Comment}
	batch, err := s.workflow.DecideReview(batchID(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) IssueCertificateHandler(w http.ResponseWriter, r *http.Request) {
	var req metaRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	batch, err := s.workflow.IssueCertificate(batchID(r), commandMeta(r, req.ExpectedVersion, req.Actor))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) VerifyCertificateHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.workflow.VerifyCertificate(batchID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
