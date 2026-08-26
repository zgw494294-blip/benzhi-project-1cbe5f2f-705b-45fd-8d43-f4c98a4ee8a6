package domain

import (
	"fmt"
	"strings"
	"time"
)

func (b *DigitizationBatch) reviewReadinessProblems() []string {
	problems := []string{}
	if b.State != StateQualityReady && b.State != StateRemediation {
		problems = append(problems, "批次不处于待提交或整改状态")
	}
	if b.QualityRunAt == nil {
		problems = append(problems, "尚未运行质量检测")
	}
	if len(b.LatestRuns()) != b.PlannedSegmentCount() {
		problems = append(problems, "冻结计划中的节目段尚未全部采集")
	}
	if len(b.OpenBlockingFindings()) > 0 {
		problems = append(problems, "仍存在未关闭的阻断缺陷")
	}
	for _, finding := range b.Findings {
		if finding.Severity == SeverityWarning && finding.Status == FindingOpen {
			problems = append(problems, "仍存在待确认的非阻断质量告警")
			break
		}
	}
	return problems
}

func (b *DigitizationBatch) OpenRemediations() []RemediationItem {
	items := []RemediationItem{}
	for _, item := range b.Remediations {
		if item.Status == RemediationOpen {
			items = append(items, item)
		}
	}
	return items
}

func (b *DigitizationBatch) SubmissionProblems() []string {
	problems := b.reviewReadinessProblems()
	if b.State == StateRemediation {
		for _, item := range b.OpenRemediations() {
			problems = append(problems, fmt.Sprintf("整改项 %s（%s）尚未提交处置证据", item.ID, item.ReasonCode))
		}
	}
	return problems
}

func (b *DigitizationBatch) SubmitReview(now time.Time) error {
	return b.SubmitReviewWithRemediations(nil, b.Operator, now)
}

func (b *DigitizationBatch) SubmitReviewWithRemediations(resolutions []RemediationResolution, actor string, now time.Time) error {
	if problems := b.reviewReadinessProblems(); len(problems) > 0 {
		return StateError("不能提交复核：%s", strings.Join(problems, "；"))
	}
	if b.State == StateQualityReady {
		if len(resolutions) > 0 {
			return Invalid("首次提交复核不能包含整改项")
		}
	} else {
		if err := b.validateAndCloseRemediations(resolutions, actor, now); err != nil {
			return err
		}
	}
	b.SubmissionNo++
	b.State = StateInReview
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) validateAndCloseRemediations(resolutions []RemediationResolution, actor string, now time.Time) error {
	open := map[string]*RemediationItem{}
	for i := range b.Remediations {
		item := &b.Remediations[i]
		if item.Status == RemediationOpen {
			open[item.ID] = item
		}
	}
	problems := []string{}
	provided := map[string]bool{}
	for i, resolution := range resolutions {
		item, ok := open[resolution.ItemID]
		if !ok {
			problems = append(problems, fmt.Sprintf("第 %d 项引用未知或已关闭的整改项 %s", i+1, resolution.ItemID))
			continue
		}
		if provided[resolution.ItemID] {
			problems = append(problems, fmt.Sprintf("整改项 %s 重复提交", resolution.ItemID))
			continue
		}
		provided[resolution.ItemID] = true
		if strings.TrimSpace(resolution.Description) == "" {
			problems = append(problems, fmt.Sprintf("整改项 %s 的处置说明不能为空", item.ID))
		}
		if err := b.validateEvidence(item, resolution.Evidence); err != nil {
			problems = append(problems, fmt.Sprintf("整改项 %s：%v", item.ID, err))
		}
	}
	for id := range open {
		if !provided[id] {
			problems = append(problems, "遗漏整改项 "+id)
		}
	}
	if len(problems) > 0 {
		return Invalid("整改项校验失败：%s", strings.Join(problems, "；"))
	}
	closedAt := now.UTC()
	for _, resolution := range resolutions {
		item := open[resolution.ItemID]
		evidence := resolution.Evidence
		item.Status = RemediationClosed
		item.Resolution = strings.TrimSpace(resolution.Description)
		item.Evidence = &evidence
		item.ClosedBy = strings.TrimSpace(actor)
		item.ClosedAt = &closedAt
	}
	return nil
}

func (b *DigitizationBatch) validateEvidence(item *RemediationItem, evidence BusinessEvidenceReference) error {
	if strings.TrimSpace(evidence.Type) == "" || strings.TrimSpace(evidence.ID) == "" {
		return Invalid("业务证据类型和 ID 不能为空")
	}
	if evidence.BatchID != b.ID {
		return Invalid("业务证据不能跨批次引用")
	}
	if evidence.SubmissionNo != item.SubmissionNo {
		return Invalid("业务证据不能跨复核轮次引用")
	}
	switch evidence.Type {
	case "capture_run":
		_, err := b.Capture(evidence.ID)
		return err
	case "finding":
		_, err := b.Finding(evidence.ID)
		return err
	case "carrier_inspection":
		for _, carrier := range b.Carriers {
			for _, inspection := range carrier.InspectionHistory {
				if inspection.ID == evidence.ID {
					return nil
				}
			}
		}
	case "risk_treatment":
		for _, carrier := range b.Carriers {
			for _, treatment := range carrier.RiskTreatments {
				if treatment.ID == evidence.ID {
					return nil
				}
			}
		}
	case "review_decision":
		for _, review := range b.Reviews {
			if review.ID == evidence.ID && review.SubmissionNo == item.SubmissionNo {
				return nil
			}
		}
	default:
		return Invalid("未知业务证据类型 %s", evidence.Type)
	}
	return NotFound("业务证据 %s/%s 不存在", evidence.Type, evidence.ID)
}

func (b *DigitizationBatch) DecideReview(d ReviewDecision, now time.Time) error {
	return b.DecideReviewWithRemediations(d, nil, now)
}

func (b *DigitizationBatch) DecideReviewWithRemediations(d ReviewDecision, items []RemediationItem, now time.Time) error {
	if b.State != StateInReview {
		return StateError("批次不在复核中")
	}
	if d.Decision != DecisionReturned && d.Decision != DecisionApproved {
		return Invalid("复核结论无效")
	}
	if strings.TrimSpace(d.Reviewer) == "" || d.Reviewer != b.Reviewer {
		return Invalid("只能由指定复核员作出决定")
	}
	if d.Decision == DecisionReturned {
		if len(d.Reasons) == 0 || len(items) != len(d.Reasons) {
			return Invalid("退回必须为每个原因提供问题说明和指定责任人")
		}
		seen := map[string]bool{}
		for i, reason := range d.Reasons {
			code := strings.TrimSpace(reason.ReasonCode)
			if code == "" || strings.TrimSpace(reason.ProblemDescription) == "" || strings.TrimSpace(reason.Assignee) == "" {
				return Invalid("退回原因第 %d 项的原因代码、问题说明和责任人不能为空", i+1)
			}
			if seen[code] {
				return Invalid("退回原因代码 %s 重复", code)
			}
			seen[code] = true
		}
	} else {
		if len(d.Reasons) > 0 || len(items) > 0 {
			return Invalid("复核通过不能创建整改项")
		}
		if len(b.OpenBlockingFindings()) > 0 {
			return StateError("存在阻断缺陷，不能通过")
		}
	}
	d.BatchID = b.ID
	d.SubmissionNo = b.SubmissionNo
	d.DecidedAt = now.UTC()
	d.ReasonCodes = make([]string, 0, len(d.Reasons))
	for _, reason := range d.Reasons {
		d.ReasonCodes = append(d.ReasonCodes, strings.TrimSpace(reason.ReasonCode))
	}
	b.Reviews = append(b.Reviews, d)
	if d.Decision == DecisionReturned {
		for i := range items {
			items[i].BatchID = b.ID
			items[i].SubmissionNo = b.SubmissionNo
			items[i].ReasonCode = strings.TrimSpace(d.Reasons[i].ReasonCode)
			items[i].ProblemDescription = strings.TrimSpace(d.Reasons[i].ProblemDescription)
			items[i].Assignee = strings.TrimSpace(d.Reasons[i].Assignee)
			items[i].CreatedAt = now.UTC()
			items[i].Status = RemediationOpen
		}
		b.Remediations = append(b.Remediations, items...)
		b.State = StateRemediation
	} else {
		b.State = StateApproved
	}
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) Seal(c AcceptanceCertificate, now time.Time) error {
	if b.State != StateApproved {
		return StateError("只有复核通过的批次才能封存")
	}
	if b.Certificate != nil {
		return StateError("验收凭据已经签发")
	}
	c.BatchID = b.ID
	b.Certificate = &c
	b.State = StateSealed
	b.Touch(now)
	return nil
}
