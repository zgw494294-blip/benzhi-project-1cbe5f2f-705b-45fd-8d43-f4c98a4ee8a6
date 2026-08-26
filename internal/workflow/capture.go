package workflow

import (
	"strings"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
)

func (s *Service) AddCaptureRun(batchID string, cmd CaptureCommand) (*domain.DigitizationBatch, error) {
	runs := cmd.Runs
	command := "add_capture_runs"
	if len(runs) == 0 {
		runs = []domain.CaptureRun{cmd.Run}
		command = "add_capture_run"
	}
	batch, replay, err := s.load(batchID, command, cmd.Meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	for i := range runs {
		if runs[i].ID != "" {
			continue
		}
		runs[i].ID, err = randomID("run")
		if err != nil {
			return nil, err
		}
	}
	if err := batch.AddCaptureRuns(runs, s.now()); err != nil {
		return nil, err
	}
	details := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		details = append(details, map[string]any{"runID": run.ID, "carrierID": run.CarrierID, "segmentID": run.SegmentID, "attempt": run.Attempt, "outputFile": run.OutputFile})
	}
	return s.commit(batch, cmd.Meta, command, map[string]any{"count": len(runs), "runs": details}, false)
}

func (s *Service) RunQuality(batchID string, meta CommandMeta) (*domain.DigitizationBatch, error) {
	batch, replay, err := s.load(batchID, "run_quality", meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	findings, err := s.quality.EvaluateBatch(batch)
	if err != nil {
		return nil, err
	}
	if err := batch.ReplaceFindings(findings, quality.RuleSetVersion, s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, meta, "run_quality", quality.Summarize(findings), false)
}

func (s *Service) ResolveFinding(batchID string, cmd ResolveFindingCommand) (*domain.DigitizationBatch, error) {
	action := strings.TrimSpace(cmd.Action)
	if action == "" {
		action = "replace"
	}
	command := "resolve_finding"
	if action == "accept_warning" {
		command = "accept_warning"
	}
	batch, replay, err := s.load(batchID, command, cmd.Meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	if action == "accept_warning" {
		if err := batch.AcceptWarning(cmd.FindingID, cmd.ReasonCode, cmd.Description, cmd.Meta.Actor, s.now()); err != nil {
			return nil, err
		}
		finding, _ := batch.Finding(cmd.FindingID)
		return s.commit(batch, cmd.Meta, command, map[string]any{"findingID": cmd.FindingID, "reasonCode": cmd.ReasonCode, "evidenceIdentity": finding.WarningAcceptance.EvidenceIdentity}, false)
	}
	if action != "replace" {
		return nil, domain.Invalid("缺陷处置动作 %s 无效", action)
	}
	f, err := batch.Finding(cmd.FindingID)
	if err != nil {
		return nil, err
	}
	replacement, err := batch.Capture(cmd.ReplacementRunID)
	if err != nil {
		return nil, err
	}
	canClose, comparison, err := s.quality.EvaluateReplacement(batch, *f, *replacement)
	if err != nil {
		return nil, err
	}
	if err := batch.ResolveFinding(cmd.FindingID, cmd.Resolution, cmd.ReplacementRunID, canClose, s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, cmd.Meta, "resolve_finding", map[string]any{"findingID": cmd.FindingID, "replacementRunID": cmd.ReplacementRunID, "closed": canClose, "comparison": comparison}, false)
}
