package quality

import (
	"sort"
	"tape-preservation-gate/internal/domain"
)

func (e *Engine) EvaluateBatch(batch *domain.DigitizationBatch) ([]domain.QualityFinding, error) {
	latest := batch.LatestRuns()
	sort.Slice(latest, func(i, j int) bool {
		if latest[i].CarrierID != latest[j].CarrierID {
			return latest[i].CarrierID < latest[j].CarrierID
		}
		return latest[i].SegmentID < latest[j].SegmentID
	})
	findings := []domain.QualityFinding{}
	for _, run := range latest {
		segment, err := findSegment(batch, run.CarrierID, run.SegmentID)
		if err != nil {
			return nil, err
		}
		for _, rule := range e.rules {
			if result := rule.Evaluate(batch.TargetProfile, segment, run); result != nil {
				findings = append(findings, *result)
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].CaptureRunID != findings[j].CaptureRunID {
			return findings[i].CaptureRunID < findings[j].CaptureRunID
		}
		return findings[i].RuleCode < findings[j].RuleCode
	})
	return findings, nil
}

func findSegment(batch *domain.DigitizationBatch, carrierID, segmentID string) (domain.ProgramSegment, error) {
	c, err := batch.Carrier(carrierID)
	if err != nil {
		return domain.ProgramSegment{}, err
	}
	for _, s := range c.Segments {
		if s.ID == segmentID {
			return s, nil
		}
	}
	return domain.ProgramSegment{}, domain.NotFound("节目段 %s 不存在", segmentID)
}

func (e *Engine) EvaluateReplacement(batch *domain.DigitizationBatch, old domain.QualityFinding, replacement domain.CaptureRun) (bool, map[string]string, error) {
	segment, err := findSegment(batch, replacement.CarrierID, replacement.SegmentID)
	if err != nil {
		return false, nil, err
	}
	for _, rule := range e.rules {
		if rule.Code() != old.RuleCode {
			continue
		}
		result := rule.Evaluate(batch.TargetProfile, segment, replacement)
		if result == nil {
			return true, map[string]string{"comparison": "替代轮次已满足原规则", "replacementRunID": replacement.ID}, nil
		}
		return false, result.Evidence, nil
	}
	return false, nil, domain.Invalid("未知质量规则 %s", old.RuleCode)
}
