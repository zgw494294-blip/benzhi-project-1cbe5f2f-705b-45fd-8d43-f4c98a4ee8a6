package certificate_quality_integrity_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
	"tape-preservation-gate/internal/workflow"
)

func TestCertificateDetectsQualityConclusionTampering(t *testing.T) {
	dir := t.TempDir()
	repo, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := domain.NewBatch("batch", "质量结论完整性", "operator", "reviewer", domain.DefaultTargetProfile(), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	batch.State = domain.StateApproved
	created, err := repo.Commit(store.CommitRequest{Batch: batch, ExpectedVersion: 0, IdempotencyKey: "create", Command: "create", Actor: "operator", Result: batch, Details: map[string]string{"step": "approved"}, Creating: true})
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(repo, quality.NewEngine())
	sealed, err := service.IssueCertificate(batch.ID, workflow.CommandMeta{ExpectedVersion: created.Batch.Version, IdempotencyKey: "certificate", Actor: "reviewer"})
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath := filepath.Join(dir, "snapshot.json")
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot store.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	snapshot.Batches[batch.ID].Findings = append(snapshot.Batches[batch.ID].Findings, domain.QualityFinding{
		ID: "tampered-finding", BatchID: batch.ID, CaptureRunID: "tampered-run", RuleCode: "DROPOUT", RuleVersion: quality.RuleSetVersion, Severity: domain.SeverityBlocking, Status: domain.FindingOpen,
	})
	data, err = json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	verification, err := workflow.New(reopened, quality.NewEngine()).VerifyCertificate(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Valid {
		t.Fatalf("质量结论被修改后凭据仍被判定有效: certificate=%s version=%d", sealed.Certificate.ID, sealed.Version)
	}
}
