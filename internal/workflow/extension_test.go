package workflow

import (
	"strings"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
	"testing"
)

func TestBatchCaptureReplayAndPlanSummary(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, quality.NewEngine())
	meta := func(v uint64, key string) CommandMeta {
		return CommandMeta{ExpectedVersion: v, IdempotencyKey: key, Actor: "op"}
	}
	b, err := svc.CreateBatch(CreateBatchCommand{ID: "b", Title: "扩展流程", Operator: "op", Reviewer: "rv", Meta: meta(0, "01")})
	if err != nil {
		t.Fatal(err)
	}
	carrier := domain.TapeCarrier{ID: "c", ArchiveCode: "A", Format: "reel", DurationMillis: 30000, Segments: []domain.ProgramSegment{{ID: "s2", Title: "二", StartMillis: 20000, DurationMillis: 5000}, {ID: "s1", Title: "一", StartMillis: 0, DurationMillis: 10000}}}
	b, err = svc.AddCarrier("b", AddCarrierCommand{Meta: meta(b.Version, "02"), Carrier: carrier})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.InspectCarrier("b", InspectCarrierCommand{Meta: meta(b.Version, "03"), CarrierID: "c", Inspection: domain.CarrierInspection{AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.FreezePlan("b", meta(b.Version, "04"))
	if err != nil {
		t.Fatal(err)
	}
	base := domain.CaptureRun{CarrierID: "c", Attempt: 1, EquipmentChain: []string{"deck"}, ChecksumSHA256: strings.Repeat("a", 64), Measurements: domain.SignalMeasurements{PeakDBFS: -3}}
	first, second := base, base
	first.ID, first.SegmentID, first.OutputFile, first.Measurements.MeasuredDuration = "r1", "s1", "one.wav", 10000
	second.ID, second.SegmentID, second.OutputFile, second.Measurements.MeasuredDuration = "r2", "s2", "two.wav", 5000
	before := b.Version
	cmd := CaptureCommand{Meta: meta(before, "05"), Runs: []domain.CaptureRun{first, second}}
	b, err = svc.AddCaptureRun("b", cmd)
	if err != nil {
		t.Fatal(err)
	}
	if b.Version != before+1 || len(b.CaptureRuns) != 2 {
		t.Fatal("批量采集没有在一个版本中提交")
	}
	replay, err := svc.AddCaptureRun("b", cmd)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Version != b.Version || len(replay.CaptureRuns) != 2 {
		t.Fatal("批量采集幂等重放结果不稳定")
	}
	view, err := svc.BatchView("b")
	if err != nil {
		t.Fatal(err)
	}
	if view.Plan.SegmentCount != 2 || view.Plan.PlannedDurationMillis != 15000 || view.Plan.GapDurationMillis != 15000 {
		t.Fatalf("计划汇总错误: %+v", view.Plan)
	}
}

func TestReviewRemediationAtomicClosure(t *testing.T) {
	repo, _ := store.Open(t.TempDir())
	svc := New(repo, quality.NewEngine())
	meta := func(v uint64, key, actor string) CommandMeta {
		return CommandMeta{ExpectedVersion: v, IdempotencyKey: key, Actor: actor}
	}
	b, _ := svc.CreateBatch(CreateBatchCommand{ID: "b", Title: "整改测试", Operator: "op", Reviewer: "rv", Meta: meta(0, "01", "op")})
	carrier := domain.TapeCarrier{ID: "c", ArchiveCode: "A", Format: "reel", DurationMillis: 10000, Segments: []domain.ProgramSegment{{ID: "s", Title: "节目", DurationMillis: 10000}}}
	b, _ = svc.AddCarrier("b", AddCarrierCommand{Meta: meta(b.Version, "02", "op"), Carrier: carrier})
	b, _ = svc.InspectCarrier("b", InspectCarrierCommand{Meta: meta(b.Version, "03", "op"), CarrierID: "c", Inspection: domain.CarrierInspection{AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}})
	b, _ = svc.FreezePlan("b", meta(b.Version, "04", "op"))
	run := domain.CaptureRun{ID: "r", CarrierID: "c", SegmentID: "s", Attempt: 1, EquipmentChain: []string{"deck"}, OutputFile: "a.wav", ChecksumSHA256: strings.Repeat("a", 64), Measurements: domain.SignalMeasurements{PeakDBFS: -3, MeasuredDuration: 10000}}
	b, _ = svc.AddCaptureRun("b", CaptureCommand{Meta: meta(b.Version, "05", "op"), Run: run})
	b, _ = svc.RunQuality("b", meta(b.Version, "06", "op"))
	b, _ = svc.SubmitReview("b", meta(b.Version, "07", "op"))
	reasons := []domain.ReviewReason{{ReasonCode: "METADATA_FIX", ProblemDescription: "补充元数据依据", Assignee: "op"}, {ReasonCode: "SIGNAL_RECHECK", ProblemDescription: "复核采集证据", Assignee: "op"}}
	b, err := svc.DecideReview("b", ReviewCommand{Meta: meta(b.Version, "08", "rv"), Decision: domain.DecisionReturned, Reasons: reasons, Comment: "退回"})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Remediations) != 2 || b.State != domain.StateRemediation {
		t.Fatal("退回没有生成逐项整改清单")
	}
	version, submission := b.Version, b.SubmissionNo
	one := domain.RemediationResolution{ItemID: b.Remediations[0].ID, Description: "已补充", Evidence: domain.BusinessEvidenceReference{Type: "capture_run", ID: "r", BatchID: "b", SubmissionNo: submission}}
	if _, err := svc.SubmitReviewWithRemediations("b", SubmitReviewCommand{Meta: meta(version, "09", "op"), Remediations: []domain.RemediationResolution{one}}); err == nil {
		t.Fatal("遗漏整改项时不应提交")
	}
	unchanged, _ := svc.GetBatch("b")
	if unchanged.Version != version || unchanged.SubmissionNo != submission || unchanged.Remediations[0].Status != domain.RemediationOpen {
		t.Fatal("整改提交失败后发生部分更新")
	}
	two := domain.RemediationResolution{ItemID: b.Remediations[1].ID, Description: "已复核", Evidence: domain.BusinessEvidenceReference{Type: "capture_run", ID: "r", BatchID: "b", SubmissionNo: submission}}
	b, err = svc.SubmitReviewWithRemediations("b", SubmitReviewCommand{Meta: meta(version, "10", "op"), Remediations: []domain.RemediationResolution{one, two}})
	if err != nil {
		t.Fatal(err)
	}
	if b.SubmissionNo != submission+1 || b.State != domain.StateInReview || b.Remediations[0].Status != domain.RemediationClosed || b.Remediations[1].Status != domain.RemediationClosed {
		t.Fatal("整改项与新复核提交未原子闭环")
	}
}
