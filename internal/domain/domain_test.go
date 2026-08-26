package domain

import (
	"strings"
	"testing"
	"time"
)

func testBatch(t *testing.T) *DigitizationBatch {
	t.Helper()
	b, err := NewBatch("b1", "测试批次", "op", "reviewer", DefaultTargetProfile(), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestUnsafeCarrierBlocksFreeze(t *testing.T) {
	b := testBatch(t)
	c := TapeCarrier{ID: "c1", ArchiveCode: "A1", Format: "reel", DurationMillis: 10000, Segments: []ProgramSegment{{ID: "s1", Title: "节目", DurationMillis: 10000}}}
	if err := b.AddCarrier(c, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	inspection := CarrierInspection{AppearanceOK: true, Mold: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}
	if err := b.InspectCarrier("c1", inspection, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.FreezePlan(time.Unix(4, 0)); err == nil || !strings.Contains(err.Error(), "未通过") {
		t.Fatalf("预期载体安全阻断，得到 %v", err)
	}
}

func TestCaptureReferenceAndSubmission(t *testing.T) {
	b := testBatch(t)
	c := TapeCarrier{ID: "c1", ArchiveCode: "A1", Format: "reel", DurationMillis: 10000, Segments: []ProgramSegment{{ID: "s1", Title: "节目", DurationMillis: 10000}}}
	if err := b.AddCarrier(c, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.InspectCarrier("c1", CarrierInspection{AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.FreezePlan(time.Unix(4, 0)); err != nil {
		t.Fatal(err)
	}
	run := CaptureRun{ID: "r1", CarrierID: "c1", SegmentID: "s1", Attempt: 1, EquipmentChain: []string{"deck"}, OutputFile: "a.wav", ChecksumSHA256: strings.Repeat("a", 64), Measurements: SignalMeasurements{MeasuredDuration: 10000}}
	if err := b.AddCaptureRun(run, time.Unix(5, 0)); err != nil {
		t.Fatal(err)
	}
	if err := b.ReplaceFindings(nil, "rules/1", time.Unix(6, 0)); err != nil {
		t.Fatal(err)
	}
	if got := b.SubmissionProblems(); len(got) != 0 {
		t.Fatalf("意外完整性问题: %v", got)
	}
	if err := b.SubmitReview(time.Unix(7, 0)); err != nil {
		t.Fatal(err)
	}
	if b.State != StateInReview || b.SubmissionNo != 1 {
		t.Fatalf("状态迁移失败: %s/%d", b.State, b.SubmissionNo)
	}
}

func TestManifestStableAcrossInsertionOrder(t *testing.T) {
	b := testBatch(t)
	b.CaptureRuns = []CaptureRun{{ID: "r2", CarrierID: "c2", SegmentID: "s2", Attempt: 1, OutputFile: "b.wav", ChecksumSHA256: strings.Repeat("b", 64)}, {ID: "r1", CarrierID: "c1", SegmentID: "s1", Attempt: 1, OutputFile: "a.wav", ChecksumSHA256: strings.Repeat("a", 64)}}
	d1, err := StableDigest(b.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	b.CaptureRuns[0], b.CaptureRuns[1] = b.CaptureRuns[1], b.CaptureRuns[0]
	d2, err := StableDigest(b.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatalf("清单摘要不稳定: %s != %s", d1, d2)
	}
}
