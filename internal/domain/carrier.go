package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	RiskAppearance = "APPEARANCE_DAMAGE"
	RiskMold       = "MOLD"
	RiskSticky     = "STICKY_SHED"
	RiskHub        = "HUB_DAMAGE"
	RiskLeader     = "LEADER_DAMAGE"
)

func (b *DigitizationBatch) AddCarrier(c TapeCarrier, now time.Time) error {
	if err := b.EnsureMutable(); err != nil {
		return err
	}
	if b.State != StateDraft {
		return StateError("采集计划冻结后不能新增或改写节目段")
	}
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.ArchiveCode) == "" || strings.TrimSpace(c.Format) == "" || c.DurationMillis <= 0 {
		return Invalid("载体身份、格式和时长必须完整")
	}
	for _, old := range b.Carriers {
		if old.ID == c.ID || old.ArchiveCode == c.ArchiveCode {
			return Invalid("载体 ID 或档案号重复")
		}
	}
	if len(c.Segments) == 0 {
		return Invalid("载体至少需要一个节目段")
	}
	seen := map[string]bool{}
	for i := range c.Segments {
		s := &c.Segments[i]
		s.ID = strings.TrimSpace(s.ID)
		s.Title = strings.TrimSpace(s.Title)
		if s.ID == "" {
			return Invalid("节目段第 %d 行：ID 不能为空", i+1)
		}
		if s.Title == "" {
			return Invalid("节目段第 %d 行：标题不能为空", i+1)
		}
		if s.StartMillis < 0 {
			return Invalid("节目段第 %d 行：起始时间不能为负数", i+1)
		}
		if s.DurationMillis <= 0 {
			return Invalid("节目段第 %d 行：时长必须为正数", i+1)
		}
		if seen[s.ID] {
			return Invalid("节目段第 %d 行：ID %s 重复", i+1, s.ID)
		}
		if s.StartMillis > c.DurationMillis-s.DurationMillis {
			return Invalid("节目段第 %d 行：结束时间 %dms 超出载体总时长 %dms", i+1, s.StartMillis+s.DurationMillis, c.DurationMillis)
		}
		seen[s.ID] = true
	}
	sort.SliceStable(c.Segments, func(i, j int) bool { return c.Segments[i].StartMillis < c.Segments[j].StartMillis })
	for i := 1; i < len(c.Segments); i++ {
		previous, current := c.Segments[i-1], c.Segments[i]
		previousEnd := previous.StartMillis + previous.DurationMillis
		if current.StartMillis < previousEnd {
			return Invalid("节目段 %s 与节目段 %s 时间重叠（%dms < %dms）", current.ID, previous.ID, current.StartMillis, previousEnd)
		}
	}
	c.BatchID = b.ID
	c.ArchiveCode = strings.TrimSpace(c.ArchiveCode)
	c.Format = strings.TrimSpace(c.Format)
	c.CaptureEligibility = EligibilityPending
	c.InspectionHistory = []CarrierInspection{}
	c.RiskTreatments = []CarrierRiskTreatment{}
	b.Carriers = append(b.Carriers, c)
	b.Touch(now)
	return nil
}

func InspectionRiskCodes(in CarrierInspection) []string {
	risks := []string{}
	if !in.AppearanceOK {
		risks = append(risks, RiskAppearance)
	}
	if in.Mold {
		risks = append(risks, RiskMold)
	}
	if in.Sticky {
		risks = append(risks, RiskSticky)
	}
	if !in.HubOK {
		risks = append(risks, RiskHub)
	}
	if !in.LeaderOK {
		risks = append(risks, RiskLeader)
	}
	return risks
}

func (c *TapeCarrier) latestInspection() *CarrierInspection {
	if len(c.InspectionHistory) == 0 {
		return nil
	}
	return &c.InspectionHistory[len(c.InspectionHistory)-1]
}

func (c *TapeCarrier) unresolvedLatestRisks() []string {
	latest := c.latestInspection()
	if latest == nil || latest.Eligibility != EligibilityBlocked {
		return nil
	}
	treated := map[string]bool{}
	for _, treatment := range c.RiskTreatments {
		if treatment.InspectionID == latest.ID {
			for _, code := range treatment.RiskCodes {
				treated[code] = true
			}
		}
	}
	result := []string{}
	for _, code := range latest.RiskCodes {
		if !treated[code] {
			result = append(result, code)
		}
	}
	return result
}

func (b *DigitizationBatch) InspectCarrier(carrierID string, in CarrierInspection, now time.Time) error {
	if err := b.EnsureMutable(); err != nil {
		return err
	}
	if b.State != StateDraft {
		return StateError("采集计划冻结后不能修改盘前检查")
	}
	if strings.TrimSpace(in.CheckedBy) == "" {
		return Invalid("检查人不能为空")
	}
	c, err := b.Carrier(carrierID)
	if err != nil {
		return err
	}
	if latest := c.latestInspection(); latest != nil && latest.Eligibility == EligibilityBlocked {
		if risks := c.unresolvedLatestRisks(); len(risks) > 0 {
			return StateError("载体 %s 的风险尚未完成处置：%s", c.ArchiveCode, strings.Join(risks, ", "))
		}
	}
	if strings.TrimSpace(in.ID) == "" {
		in.ID = carrierID + "-inspection-" + fmt.Sprint(len(c.InspectionHistory)+1)
	}
	for _, old := range c.InspectionHistory {
		if old.ID == in.ID {
			return Invalid("检查记录 ID %s 重复", in.ID)
		}
	}
	in.CheckedBy = strings.TrimSpace(in.CheckedBy)
	in.CheckedAt = now.UTC()
	in.Sequence = len(c.InspectionHistory) + 1
	in.RiskCodes = InspectionRiskCodes(in)
	in.Eligibility = EligibilityAllowed
	if len(in.RiskCodes) > 0 {
		in.Eligibility = EligibilityBlocked
	}
	c.InspectionHistory = append(c.InspectionHistory, in)
	c.Inspection = in
	c.CaptureEligibility = in.Eligibility
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) TreatCarrierRisks(carrierID string, treatment CarrierRiskTreatment, now time.Time) error {
	if b.State != StateDraft {
		return StateError("只能在草稿阶段登记盘前风险处置")
	}
	if strings.TrimSpace(treatment.ID) == "" || strings.TrimSpace(treatment.Method) == "" || strings.TrimSpace(treatment.Description) == "" || strings.TrimSpace(treatment.ExecutedBy) == "" {
		return Invalid("处置方式、处置说明和执行人不能为空")
	}
	if len(treatment.RiskCodes) == 0 {
		return Invalid("风险处置至少需要一个风险代码")
	}
	c, err := b.Carrier(carrierID)
	if err != nil {
		return err
	}
	for _, old := range c.RiskTreatments {
		if old.ID == treatment.ID {
			return Invalid("风险处置 ID %s 重复", treatment.ID)
		}
	}
	latest := c.latestInspection()
	if latest == nil || latest.Eligibility != EligibilityBlocked {
		return StateError("载体 %s 当前没有可处置的阻断风险", c.ArchiveCode)
	}
	unresolved := map[string]bool{}
	for _, code := range c.unresolvedLatestRisks() {
		unresolved[code] = true
	}
	seen := map[string]bool{}
	for i, code := range treatment.RiskCodes {
		code = strings.TrimSpace(code)
		if code == "" || !unresolved[code] {
			return Invalid("风险代码 %s 不属于上一轮未闭环风险", code)
		}
		if seen[code] {
			return Invalid("风险代码 %s 重复", code)
		}
		treatment.RiskCodes[i] = code
		seen[code] = true
	}
	treatment.InspectionID = latest.ID
	treatment.Method = strings.TrimSpace(treatment.Method)
	treatment.Description = strings.TrimSpace(treatment.Description)
	treatment.ExecutedBy = strings.TrimSpace(treatment.ExecutedBy)
	treatment.ExecutedAt = now.UTC()
	c.RiskTreatments = append(c.RiskTreatments, treatment)
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) FreezeProblems() []string {
	problems := []string{}
	if len(b.Carriers) == 0 {
		return []string{"没有可冻结的载体"}
	}
	for i := range b.Carriers {
		c := &b.Carriers[i]
		latest := c.latestInspection()
		if latest == nil {
			problems = append(problems, "载体 "+c.ArchiveCode+" 尚未完成盘前检查")
			continue
		}
		if latest.Eligibility == EligibilityBlocked {
			if risks := c.unresolvedLatestRisks(); len(risks) > 0 {
				problems = append(problems, "载体 "+c.ArchiveCode+" 仍有未处置风险："+strings.Join(risks, ", "))
			} else {
				problems = append(problems, "载体 "+c.ArchiveCode+" 风险处置后尚未复检通过")
			}
		}
	}
	return problems
}

func (b *DigitizationBatch) FreezePlan(now time.Time) error {
	if b.State != StateDraft {
		return StateError("当前状态不能冻结采集计划")
	}
	if problems := b.FreezeProblems(); len(problems) > 0 {
		return StateError("冻结预检未通过：%s", strings.Join(problems, "；"))
	}
	b.State = StatePlanFrozen
	t := now.UTC()
	b.PlanFrozenAt = &t
	b.Touch(now)
	return nil
}
