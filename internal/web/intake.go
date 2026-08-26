package web

import (
	"net/http"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/workflow"
)

type addCarrierRequest struct {
	ExpectedVersion uint64             `json:"expectedVersion"`
	Actor           string             `json:"actor"`
	Carrier         domain.TapeCarrier `json:"carrier"`
}

type inspectCarrierRequest struct {
	ExpectedVersion uint64                      `json:"expectedVersion"`
	Actor           string                      `json:"actor"`
	Action          string                      `json:"action"`
	CarrierID       string                      `json:"carrierID"`
	Inspection      domain.CarrierInspection    `json:"inspection"`
	Treatment       domain.CarrierRiskTreatment `json:"treatment"`
}

type metaRequest struct {
	ExpectedVersion uint64 `json:"expectedVersion"`
	Actor           string `json:"actor"`
}

func (s *Server) AddCarrierHandler(w http.ResponseWriter, r *http.Request) {
	var req addCarrierRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if len(req.Carrier.Segments) == 0 || len(req.Carrier.Segments) > 100 {
		writeError(w, domain.Invalid("segments 数组必须包含 1 到 100 个节目段"))
		return
	}
	batch, err := s.workflow.AddCarrier(batchID(r), workflow.AddCarrierCommand{Meta: commandMeta(r, req.ExpectedVersion, req.Actor), Carrier: req.Carrier})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) InspectCarrierHandler(w http.ResponseWriter, r *http.Request) {
	var req inspectCarrierRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	batch, err := s.workflow.InspectCarrier(batchID(r), workflow.InspectCarrierCommand{Meta: commandMeta(r, req.ExpectedVersion, req.Actor), Action: req.Action, CarrierID: req.CarrierID, Inspection: req.Inspection, Treatment: req.Treatment})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) FreezePlanHandler(w http.ResponseWriter, r *http.Request) {
	var req metaRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	batch, err := s.workflow.FreezePlan(batchID(r), commandMeta(r, req.ExpectedVersion, req.Actor))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
