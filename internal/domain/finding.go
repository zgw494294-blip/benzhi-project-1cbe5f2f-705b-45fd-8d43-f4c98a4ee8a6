package domain

import (
	"sort"
	"strings"
	"time"
)

var warningReasonCodes = map[string]bool{
	"EXPECTED_CONTENT":      true,
	"PROGRAM_CONTENT":       true,
	"SOURCE_CHARACTERISTIC": true,
	"ARCHIVAL_CONTEXT":      true,
	"TECHNICAL_LIMITATION":  true,
	"DOCUMENTED_EXCEPTION":  true,
	"LONG_SILENCE_EXPECTED": true,
}

func FindingEvidenceIdentity(f QualityFinding) string {
	key := struct {
		CaptureRunID string
		RuleCode     string
		RuleVersion  string
		StartMillis  int64
		EndMillis    int64
		Evidence     map[string]string
	}{f.CaptureRunID, f.RuleCode, f.RuleVersion, f.StartMillis, f.EndMillis, f.Evidence}
	digest, _ := StableDigest(key)
	return digest
}

func (b *DigitizationBatch) ReplaceFindings(findings []QualityFinding, ruleVersion string, now time.Time) error {
	if b.State != StatePlanFrozen && b.State != StateQualityReady && b.State != StateRemediation {
		return StateError("当前状态不能运行质量检测")
	}
	oldByIdentity := map[string]QualityFinding{}
	for _, old := range b.Findings {
		oldByIdentity[FindingEvidenceIdentity(old)] = old
	}
	newIdentities := map[string]bool{}
	for i := range findings {
		f := &findings[i]
		if _, err := b.Capture(f.CaptureRunID); err != nil {
			return err
		}
		f.BatchID = b.ID
		f.RuleVersion = ruleVersion
		identity := FindingEvidenceIdentity(*f)
		newIdentities[identity] = true
		if old, ok := oldByIdentity[identity]; ok && old.Severity == SeverityWarning && old.Status == FindingAccepted && old.WarningAcceptance != nil {
			f.Status = FindingAccepted
			f.WarningAcceptance = old.WarningAcceptance
		} else if f.Status == "" {
			f.Status = FindingOpen
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].CaptureRunID != findings[j].CaptureRunID {
			return findings[i].CaptureRunID < findings[j].CaptureRunID
		}
		if findings[i].RuleCode != findings[j].RuleCode {
			return findings[i].RuleCode < findings[j].RuleCode
		}
		return findings[i].StartMillis < findings[j].StartMillis
	})
	history := make([]QualityFinding, 0)
	for _, old := range b.Findings {
		if !newIdentities[FindingEvidenceIdentity(old)] {
			history = append(history, old)
		}
	}
	b.Findings = append(history, findings...)
	b.QualityVersion = ruleVersion
	t := now.UTC()
	b.QualityRunAt = &t
	if b.State == StatePlanFrozen {
		b.State = StateQualityReady
	}
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) AcceptWarning(id, reasonCode, description, acceptedBy string, now time.Time) error {
	if b.State != StateQualityReady && b.State != StateRemediation {
		return StateError("当前状态不能确认质量告警")
	}
	f, err := b.Finding(id)
	if err != nil {
		return err
	}
	if f.Severity != SeverityWarning {
		return StateError("blocking 缺陷不能通过告警确认绕过，必须使用定向重采")
	}
	if f.Status != FindingOpen {
		return StateError("质量告警当前不是待确认状态")
	}
	reasonCode = strings.TrimSpace(reasonCode)
	if !warningReasonCodes[reasonCode] {
		return Invalid("告警确认原因代码 %s 不在受控列表中", reasonCode)
	}
	if strings.TrimSpace(description) == "" || strings.TrimSpace(acceptedBy) == "" {
		return Invalid("告警确认说明和确认人不能为空")
	}
	f.Status = FindingAccepted
	f.WarningAcceptance = &WarningAcceptance{ReasonCode: reasonCode, Description: strings.TrimSpace(description), AcceptedBy: strings.TrimSpace(acceptedBy), AcceptedAt: now.UTC(), EvidenceIdentity: FindingEvidenceIdentity(*f)}
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) ResolveFinding(id, resolution, replacementRunID string, allowClose bool, now time.Time) error {
	if b.State != StateQualityReady && b.State != StateRemediation {
		return StateError("当前状态不能处置缺陷")
	}
	if strings.TrimSpace(resolution) == "" {
		return Invalid("处置意见不能为空")
	}
	f, err := b.Finding(id)
	if err != nil {
		return err
	}
	if f.Severity != SeverityBlocking {
		return StateError("warning 缺陷应使用告警确认动作，无需关联替代采集轮次")
	}
	if f.Status == FindingClosed {
		return StateError("缺陷已关闭")
	}
	if replacementRunID == "" {
		return Invalid("阻断缺陷必须关联定向重采轮次")
	}
	run, err := b.Capture(replacementRunID)
	if err != nil {
		return err
	}
	original, err := b.Capture(f.CaptureRunID)
	if err != nil {
		return err
	}
	if run.SupersedesRunID != original.ID {
		return Invalid("替代采集未直接关联原缺陷轮次")
	}
	f.Resolution = strings.TrimSpace(resolution)
	f.ReplacementRunID = replacementRunID
	f.Status = FindingResolved
	if allowClose {
		f.Status = FindingClosed
	}
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) OpenBlockingFindings() []QualityFinding {
	result := []QualityFinding{}
	for _, f := range b.Findings {
		if f.Severity == SeverityBlocking && f.Status != FindingClosed {
			result = append(result, f)
		}
	}
	return result
}
