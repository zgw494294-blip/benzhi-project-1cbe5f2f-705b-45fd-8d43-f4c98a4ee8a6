package workflow

import (
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
)

type BatchView struct {
	Batch          *domain.DigitizationBatch `json:"batch"`
	AllowedActions []string                  `json:"allowedActions"`
	Problems       []string                  `json:"submissionProblems"`
	Quality        quality.ResultSummary     `json:"qualitySummary"`
	Audit          []store.AuditEvent        `json:"audit"`
	Plan           PlanSummary               `json:"planSummary"`
	CapturePlan    []CaptureTargetView       `json:"capturePlan"`
	FreezeProblems []string                  `json:"freezeProblems"`
}

type SegmentRangeView struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	StartMillis    int64  `json:"startMillis"`
	EndMillis      int64  `json:"endMillis"`
	DurationMillis int64  `json:"durationMillis"`
}

type CarrierPlanSummary struct {
	CarrierID             string             `json:"carrierID"`
	ArchiveCode           string             `json:"archiveCode"`
	CarrierDurationMillis int64              `json:"carrierDurationMillis"`
	PlannedDurationMillis int64              `json:"plannedDurationMillis"`
	GapDurationMillis     int64              `json:"gapDurationMillis"`
	Segments              []SegmentRangeView `json:"segments"`
}

type PlanSummary struct {
	SegmentCount          int                  `json:"segmentCount"`
	PlannedDurationMillis int64                `json:"plannedDurationMillis"`
	GapDurationMillis     int64                `json:"gapDurationMillis"`
	Carriers              []CarrierPlanSummary `json:"carriers"`
}

type CaptureTargetView struct {
	CarrierID   string `json:"carrierID"`
	SegmentID   string `json:"segmentID"`
	ArchiveCode string `json:"archiveCode"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	LatestRunID string `json:"latestRunID,omitempty"`
	Attempt     int    `json:"attempt"`
}

func (s *Service) BatchView(id string) (BatchView, error) {
	b, err := s.repo.Get(id)
	if err != nil {
		return BatchView{}, err
	}
	plan := buildPlanSummary(b)
	return BatchView{Batch: b, AllowedActions: b.AllowedActions(), Problems: b.SubmissionProblems(), Quality: quality.Summarize(b.Findings), Audit: s.repo.AuditForBatch(id), Plan: plan, CapturePlan: buildCapturePlan(b), FreezeProblems: b.FreezeProblems()}, nil
}

func buildPlanSummary(b *domain.DigitizationBatch) PlanSummary {
	result := PlanSummary{Carriers: []CarrierPlanSummary{}}
	for _, carrier := range b.Carriers {
		item := CarrierPlanSummary{CarrierID: carrier.ID, ArchiveCode: carrier.ArchiveCode, CarrierDurationMillis: carrier.DurationMillis, Segments: []SegmentRangeView{}}
		for _, segment := range carrier.Segments {
			item.Segments = append(item.Segments, SegmentRangeView{ID: segment.ID, Title: segment.Title, StartMillis: segment.StartMillis, EndMillis: segment.StartMillis + segment.DurationMillis, DurationMillis: segment.DurationMillis})
			item.PlannedDurationMillis += segment.DurationMillis
			result.SegmentCount++
		}
		item.GapDurationMillis = carrier.DurationMillis - item.PlannedDurationMillis
		result.PlannedDurationMillis += item.PlannedDurationMillis
		result.GapDurationMillis += item.GapDurationMillis
		result.Carriers = append(result.Carriers, item)
	}
	return result
}

func buildCapturePlan(b *domain.DigitizationBatch) []CaptureTargetView {
	latest := map[string]domain.CaptureRun{}
	for _, run := range b.LatestRuns() {
		latest[run.CarrierID+"/"+run.SegmentID] = run
	}
	result := []CaptureTargetView{}
	for _, carrier := range b.Carriers {
		for _, segment := range carrier.Segments {
			item := CaptureTargetView{CarrierID: carrier.ID, SegmentID: segment.ID, ArchiveCode: carrier.ArchiveCode, Title: segment.Title, Status: "uncaptured"}
			if run, ok := latest[carrier.ID+"/"+segment.ID]; ok {
				item.Status, item.LatestRunID, item.Attempt = "captured", run.ID, run.Attempt
				if run.Attempt > 1 && (b.QualityRunAt == nil || run.CapturedAt.After(*b.QualityRunAt)) {
					item.Status = "pending_recheck"
				}
			}
			result = append(result, item)
		}
	}
	return result
}
