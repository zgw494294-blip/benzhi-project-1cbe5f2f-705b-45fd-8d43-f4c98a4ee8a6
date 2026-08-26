package workflow

import (
	"strings"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
	"testing"
)

func TestCompleteWorkflowAndCertificate(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, quality.NewEngine())
	meta := func(v uint64, k, actor string) CommandMeta {
		return CommandMeta{ExpectedVersion: v, IdempotencyKey: k, Actor: actor}
	}
	b, err := svc.CreateBatch(CreateBatchCommand{ID: "batch", Title: "流程测试", Operator: "op", Reviewer: "rv", Meta: meta(0, "01", "op")})
	if err != nil {
		t.Fatal(err)
	}
	carrier := domain.TapeCarrier{ID: "c", ArchiveCode: "A", Format: "reel", DurationMillis: 10000, Segments: []domain.ProgramSegment{{ID: "s", Title: "节目", DurationMillis: 10000}}}
	b, err = svc.AddCarrier(b.ID, AddCarrierCommand{Meta: meta(b.Version, "02", "op"), Carrier: carrier})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.InspectCarrier(b.ID, InspectCarrierCommand{Meta: meta(b.Version, "03", "op"), CarrierID: "c", Inspection: domain.CarrierInspection{AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.FreezePlan(b.ID, meta(b.Version, "04", "op"))
	if err != nil {
		t.Fatal(err)
	}
	run := domain.CaptureRun{ID: "r", CarrierID: "c", SegmentID: "s", Attempt: 1, EquipmentChain: []string{"deck"}, OutputFile: "a.wav", ChecksumSHA256: strings.Repeat("a", 64), Measurements: domain.SignalMeasurements{PeakDBFS: -3, MeasuredDuration: 10000}}
	b, err = svc.AddCaptureRun(b.ID, CaptureCommand{Meta: meta(b.Version, "05", "op"), Run: run})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.RunQuality(b.ID, meta(b.Version, "06", "op"))
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.SubmitReview(b.ID, meta(b.Version, "07", "op"))
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.DecideReview(b.ID, ReviewCommand{Meta: meta(b.Version, "08", "rv"), Decision: domain.DecisionApproved, Comment: "通过"})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.IssueCertificate(b.ID, meta(b.Version, "09", "rv"))
	if err != nil {
		t.Fatal(err)
	}
	verification, err := svc.VerifyCertificate(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Valid || b.State != domain.StateSealed {
		t.Fatalf("封存或核验失败: %+v", verification)
	}
}

func TestExpectedVersionAndIdempotency(t *testing.T) {
	repo, _ := store.Open(t.TempDir())
	svc := New(repo, quality.NewEngine())
	cmd := CreateBatchCommand{ID: "b", Title: "并发测试", Operator: "op", Reviewer: "rv", Meta: CommandMeta{IdempotencyKey: "same", Actor: "op"}}
	a, err := svc.CreateBatch(cmd)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateBatch(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if a.Version != b.Version {
		t.Fatalf("幂等结果不稳定")
	}
	_, err = svc.FreezePlan("b", CommandMeta{ExpectedVersion: 99, IdempotencyKey: "new", Actor: "op"})
	if err == nil || !IsConflict(err) {
		t.Fatalf("预期版本冲突，得到 %v", err)
	}
}

func TestBlockingFindingClosesOnlyWithQualifiedRecapture(t *testing.T) {
	repo, _ := store.Open(t.TempDir())
	svc := New(repo, quality.NewEngine())
	meta := func(v uint64, k string) CommandMeta {
		return CommandMeta{ExpectedVersion: v, IdempotencyKey: k, Actor: "op"}
	}
	b, err := svc.CreateBatch(CreateBatchCommand{ID: "repair", Title: "重采测试", Operator: "op", Reviewer: "rv", Meta: meta(0, "01")})
	if err != nil {
		t.Fatal(err)
	}
	carrier := domain.TapeCarrier{ID: "c", ArchiveCode: "A", Format: "reel", DurationMillis: 10000, Segments: []domain.ProgramSegment{{ID: "s", Title: "节目", DurationMillis: 10000}}}
	b, err = svc.AddCarrier(b.ID, AddCarrierCommand{Meta: meta(b.Version, "02"), Carrier: carrier})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.InspectCarrier(b.ID, InspectCarrierCommand{Meta: meta(b.Version, "03"), CarrierID: "c", Inspection: domain.CarrierInspection{AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "op"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.FreezePlan(b.ID, meta(b.Version, "04"))
	if err != nil {
		t.Fatal(err)
	}
	bad := domain.CaptureRun{ID: "bad", CarrierID: "c", SegmentID: "s", Attempt: 1, EquipmentChain: []string{"deck"}, OutputFile: "bad.wav", ChecksumSHA256: strings.Repeat("a", 64), Measurements: domain.SignalMeasurements{PeakDBFS: -3, DropoutMillis: 500, MeasuredDuration: 10000}}
	b, err = svc.AddCaptureRun(b.ID, CaptureCommand{Meta: meta(b.Version, "05"), Run: bad})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.RunQuality(b.ID, meta(b.Version, "06"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.OpenBlockingFindings()) != 1 {
		t.Fatalf("应产生一个阻断缺陷: %+v", b.Findings)
	}
	findingID := b.OpenBlockingFindings()[0].ID
	good := bad
	good.ID = "good"
	good.Attempt = 2
	good.OutputFile = "good.wav"
	good.ChecksumSHA256 = strings.Repeat("b", 64)
	good.SupersedesRunID = "bad"
	good.Measurements.DropoutMillis = 0
	b, err = svc.AddCaptureRun(b.ID, CaptureCommand{Meta: meta(b.Version, "07"), Run: good})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.ResolveFinding(b.ID, ResolveFindingCommand{Meta: meta(b.Version, "08"), FindingID: findingID, Resolution: "完成定向重采", ReplacementRunID: "good"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := b.Finding(findingID)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != domain.FindingClosed {
		t.Fatalf("合格替代证据未关闭缺陷: %s", f.Status)
	}
	b, err = svc.RunQuality(b.ID, meta(b.Version, "09"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.OpenBlockingFindings()) != 0 {
		t.Fatal("重新检测后仍有阻断缺陷")
	}
}
