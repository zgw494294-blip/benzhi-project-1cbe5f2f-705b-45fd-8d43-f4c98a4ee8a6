package workflow

import (
	"tape-preservation-gate/internal/domain"
)

func (s *Service) SubmitReview(batchID string, meta CommandMeta) (*domain.DigitizationBatch, error) {
	return s.SubmitReviewWithRemediations(batchID, SubmitReviewCommand{Meta: meta})
}

func (s *Service) SubmitReviewWithRemediations(batchID string, cmd SubmitReviewCommand) (*domain.DigitizationBatch, error) {
	meta := cmd.Meta
	batch, replay, err := s.load(batchID, "submit_review", meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	closedIDs := make([]string, 0, len(cmd.Remediations))
	for _, resolution := range cmd.Remediations {
		closedIDs = append(closedIDs, resolution.ItemID)
	}
	if err := batch.SubmitReviewWithRemediations(cmd.Remediations, meta.Actor, s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, meta, "submit_review", map[string]any{"submissionNo": batch.SubmissionNo, "closedRemediationIDs": closedIDs, "manifest": batch.Manifest()}, false)
}

func (s *Service) DecideReview(batchID string, cmd ReviewCommand) (*domain.DigitizationBatch, error) {
	batch, replay, err := s.load(batchID, "decide_review", cmd.Meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	id, err := randomID("review")
	if err != nil {
		return nil, err
	}
	reasons := cmd.Reasons
	items := make([]domain.RemediationItem, 0, len(reasons))
	for range reasons {
		itemID, idErr := randomID("remediation")
		if idErr != nil {
			return nil, idErr
		}
		items = append(items, domain.RemediationItem{ID: itemID})
	}
	d := domain.ReviewDecision{ID: id, Decision: cmd.Decision, ReasonCodes: cmd.ReasonCodes, Reasons: reasons, Comment: cmd.Comment, Reviewer: cmd.Meta.Actor}
	if err := batch.DecideReviewWithRemediations(d, items, s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, cmd.Meta, "decide_review", map[string]any{"decision": d, "remediationItems": items}, false)
}
