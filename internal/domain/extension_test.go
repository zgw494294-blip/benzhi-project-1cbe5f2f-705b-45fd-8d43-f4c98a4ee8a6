package domain

import (
	"strings"
	"testing"
	"time"
)

func TestTimelineNormalizationAndAtomicBatchCapture(t *testing.T) {
	b := testBatch(t)
	carrier := TapeCarrier{ID: "c1", ArchiveCode: "A1", Format: "reel", DurationMillis: 60000, Segments: []ProgramSegment{
		{ID: "s3", Title: "三", StartMillis: 40000, DurationMillis: 10000},
		{ID: "s1", Title: "一", StartMillis: 0, DurationMillis: 10000},
		{ID: "s2", Title: "二", StartMillis: 20000, DurationMillis: 10000},
	}}
	if err := b.AddCarrier(carrier, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	if got := b.Carriers[0].Segments[1].ID; got != "s2" {
		t.Fatalf("节目段未按起始时间规范化: %s", got)
	}
	if err := b.InspectCarrier("c1", CarrierInspection{ID: "i1", AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.FreezePlan(time.Unix(4, 0)); err != nil {
		t.Fatal(err)
	}
	version := b.Version
	base := CaptureRun{CarrierID: "c1", Attempt: 1, EquipmentChain: []string{"deck"}, ChecksumSHA256: strings.Repeat("a", 64), Measurements: SignalMeasurements{PeakDBFS: -3, MeasuredDuration: 10000}}
	r1, r2, r3 := base, base, base
	r1.ID, r1.SegmentID, r1.OutputFile = "r1", "s1", "one.wav"
	r2.ID, r2.SegmentID, r2.OutputFile = "r2", "s2", "one.wav"
	r3.ID, r3.SegmentID, r3.OutputFile, r3.Attempt = "r3", "s3", "three.wav", 2
	err := b.AddCaptureRuns([]CaptureRun{r1, r2, r3}, time.Unix(5, 0))
	if err == nil || !strings.Contains(err.Error(), "第 2 行") || !strings.Contains(err.Error(), "第 3 行") {
		t.Fatalf("应同时定位批量错误: %v", err)
	}
	if len(b.CaptureRuns) != 0 || b.Version != version {
		t.Fatal("失败批量请求不应产生部分更新")
	}
	r2.OutputFile, r3.Attempt = "two.wav", 1
	if err := b.AddCaptureRuns([]CaptureRun{r1, r2, r3}, time.Unix(6, 0)); err != nil {
		t.Fatal(err)
	}
	if len(b.CaptureRuns) != 3 || b.Version != version+1 {
		t.Fatalf("批量成功应只递增一次版本: runs=%d version=%d", len(b.CaptureRuns), b.Version)
	}
}

func TestRiskTreatmentAndReinspectionHistory(t *testing.T) {
	b := testBatch(t)
	carrier := TapeCarrier{ID: "c1", ArchiveCode: "A1", Format: "reel", DurationMillis: 10000, Segments: []ProgramSegment{{ID: "s1", Title: "节目", DurationMillis: 10000}}}
	if err := b.AddCarrier(carrier, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.InspectCarrier("c1", CarrierInspection{ID: "i1", AppearanceOK: true, Mold: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.InspectCarrier("c1", CarrierInspection{ID: "i2", AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}, time.Unix(4, 0)); err == nil {
		t.Fatal("未处置风险时不应允许直接复检")
	}
	if err := b.TreatCarrierRisks("c1", CarrierRiskTreatment{ID: "t1", RiskCodes: []string{RiskMold}, Method: "清洁", Description: "隔离后完成受控清洁", ExecutedBy: "op"}, time.Unix(5, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.InspectCarrier("c1", CarrierInspection{ID: "i2", AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}, time.Unix(6, 0)); err != nil {
		t.Fatal(err)
	}
	if len(b.Carriers[0].InspectionHistory) != 2 || len(b.Carriers[0].RiskTreatments) != 1 || b.Carriers[0].CaptureEligibility != EligibilityAllowed {
		t.Fatal("检查、处置、复检轨迹未完整保留")
	}
	if err := b.FreezePlan(time.Unix(7, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestWarningAcceptanceInheritance(t *testing.T) {
	b := testBatch(t)
	b.State = StateQualityReady
	b.CaptureRuns = []CaptureRun{{ID: "r1"}}
	finding := QualityFinding{ID: "f1", CaptureRunID: "r1", RuleCode: "LONG_SILENCE", RuleVersion: "rules/1", Severity: SeverityWarning, StartMillis: 0, EndMillis: 6000, Evidence: map[string]string{"actual": "6000"}, Status: FindingOpen}
	b.Findings = []QualityFinding{finding}
	if err := b.AcceptWarning("f1", "EXPECTED_CONTENT", "节目原有静默，允许保留", "op", time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	recheck := finding
	if err := b.ReplaceFindings([]QualityFinding{recheck}, "rules/1", time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	if b.Findings[0].Status != FindingAccepted || b.Findings[0].WarningAcceptance == nil {
		t.Fatal("相同关键证据重新检测后应继承告警确认")
	}
	changed := finding
	changed.Evidence = map[string]string{"actual": "7000"}
	if err := b.ReplaceFindings([]QualityFinding{changed}, "rules/1", time.Unix(4, 0)); err != nil {
		t.Fatal(err)
	}
	if b.Findings[len(b.Findings)-1].Status != FindingOpen {
		t.Fatal("关键证据变化后应产生待确认的新告警")
	}
}
