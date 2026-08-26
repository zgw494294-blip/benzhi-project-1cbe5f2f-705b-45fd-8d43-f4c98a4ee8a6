package domain

import (
	"strings"
	"time"
)

func NewBatch(id, title, operator, reviewer string, profile TargetProfile, now time.Time) (*DigitizationBatch, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" {
		return nil, Invalid("批次 ID 和标题不能为空")
	}
	if strings.TrimSpace(operator) == "" || strings.TrimSpace(reviewer) == "" {
		return nil, Invalid("操作员和复核员不能为空")
	}
	if operator == reviewer {
		return nil, Invalid("操作员与复核员必须为不同人员")
	}
	if err := ValidateTarget(profile); err != nil {
		return nil, err
	}
	now = now.UTC()
	return &DigitizationBatch{ID: id, Title: strings.TrimSpace(title), State: StateDraft, TargetProfile: profile, Operator: operator, Reviewer: reviewer, Version: 1, CreatedAt: now, UpdatedAt: now, Carriers: []TapeCarrier{}, CaptureRuns: []CaptureRun{}, Findings: []QualityFinding{}, Reviews: []ReviewDecision{}, Remediations: []RemediationItem{}}, nil
}

func ValidateTarget(p TargetProfile) error {
	if p.Codec == "" || p.SampleRate < 44100 || p.BitDepth < 16 || p.Channels < 1 {
		return Invalid("保存格式目标不完整或低于最低要求")
	}
	if p.PeakLimitDBFS > 0 || p.MaxSilenceMillis < 0 || p.MaxDropoutMillis < 0 || p.MaxTimebasePPM < 0 || p.DurationTolerance < 0 {
		return Invalid("质量阈值不合法")
	}
	return nil
}

func (b *DigitizationBatch) Touch(now time.Time) {
	b.Version++
	b.UpdatedAt = now.UTC()
}

func (b *DigitizationBatch) EnsureMutable() error {
	if b.State == StateApproved || b.State == StateSealed {
		return StateError("批次已经通过或封存，不允许修改")
	}
	return nil
}

func (b *DigitizationBatch) Carrier(id string) (*TapeCarrier, error) {
	for i := range b.Carriers {
		if b.Carriers[i].ID == id {
			return &b.Carriers[i], nil
		}
	}
	return nil, NotFound("载体 %s 不存在", id)
}

func (b *DigitizationBatch) Capture(id string) (*CaptureRun, error) {
	for i := range b.CaptureRuns {
		if b.CaptureRuns[i].ID == id {
			return &b.CaptureRuns[i], nil
		}
	}
	return nil, NotFound("采集轮次 %s 不存在", id)
}

func (b *DigitizationBatch) Finding(id string) (*QualityFinding, error) {
	for i := range b.Findings {
		if b.Findings[i].ID == id {
			return &b.Findings[i], nil
		}
	}
	return nil, NotFound("质量缺陷 %s 不存在", id)
}
