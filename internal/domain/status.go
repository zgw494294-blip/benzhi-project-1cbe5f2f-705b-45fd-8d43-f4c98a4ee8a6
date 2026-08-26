package domain

func (b *DigitizationBatch) AllowedActions() []string {
	switch b.State {
	case StateDraft:
		return []string{"登记载体与节目段", "追加盘前检查", "登记风险处置", "复检", "冻结采集计划"}
	case StatePlanFrozen:
		return []string{"批量登记采集轮次", "运行质量检测"}
	case StateQualityReady:
		return []string{"批量登记定向重采", "确认非阻断告警", "处置阻断缺陷", "重新检测", "提交复核"}
	case StateRemediation:
		return []string{"批量登记定向重采", "确认非阻断告警", "处置阻断缺陷", "关闭整改项并再次提交复核"}
	case StateInReview:
		return []string{"复核退回", "复核通过"}
	case StateApproved:
		return []string{"签发验收凭据"}
	case StateSealed:
		return []string{"核验验收凭据"}
	default:
		return []string{}
	}
}

func (b *DigitizationBatch) LatestReview() *ReviewDecision {
	if len(b.Reviews) == 0 {
		return nil
	}
	return &b.Reviews[len(b.Reviews)-1]
}
