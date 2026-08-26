package quality

import (
	"strings"
	"tape-preservation-gate/internal/domain"
	"testing"
	"time"
)

func qualityBatch() *domain.DigitizationBatch {
	b, _ := domain.NewBatch("b", "质量测试", "op", "reviewer", domain.DefaultTargetProfile(), time.Now())
	b.Carriers = []domain.TapeCarrier{{ID: "c", BatchID: "b", DurationMillis: 10000, Segments: []domain.ProgramSegment{{ID: "s", Title: "节目", DurationMillis: 10000}}, CaptureEligibility: domain.EligibilityAllowed}}
	b.State = domain.StatePlanFrozen
	return b
}

func TestEngineProducesDeterministicEvidence(t *testing.T) {
	b := qualityBatch()
	b.CaptureRuns = []domain.CaptureRun{{ID: "r", BatchID: "b", CarrierID: "c", SegmentID: "s", Attempt: 1, OutputFile: "a.wav", ChecksumSHA256: strings.Repeat("a", 64), Measurements: domain.SignalMeasurements{PeakDBFS: -0.2, SilenceMillis: 6000, DropoutMillis: 250, TimebasePPM: 70, MeasuredDuration: 12000}}}
	e := NewEngine()
	a, err := e.EvaluateBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	z, err := e.EvaluateBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 5 {
		t.Fatalf("预期 5 个信号缺陷，得到 %d", len(a))
	}
	for i := range a {
		if a[i].ID != z[i].ID || a[i].RuleCode != z[i].RuleCode {
			t.Fatalf("规则结果不确定")
		}
	}
}

func TestReplacementMustSatisfyOriginalRule(t *testing.T) {
	b := qualityBatch()
	oldRun := domain.CaptureRun{ID: "r1", BatchID: "b", CarrierID: "c", SegmentID: "s", Attempt: 1, ChecksumSHA256: strings.Repeat("a", 64), Measurements: domain.SignalMeasurements{DropoutMillis: 500, MeasuredDuration: 10000}}
	newRun := oldRun
	newRun.ID = "r2"
	newRun.Attempt = 2
	newRun.SupersedesRunID = "r1"
	newRun.Measurements.DropoutMillis = 0
	b.CaptureRuns = []domain.CaptureRun{oldRun, newRun}
	f := domain.QualityFinding{CaptureRunID: "r1", RuleCode: "DROPOUT", Severity: domain.SeverityBlocking}
	ok, evidence, err := NewEngine().EvaluateReplacement(b, f, newRun)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || evidence["replacementRunID"] != "r2" {
		t.Fatalf("有效替代证据未被接受: %v %v", ok, evidence)
	}
}
