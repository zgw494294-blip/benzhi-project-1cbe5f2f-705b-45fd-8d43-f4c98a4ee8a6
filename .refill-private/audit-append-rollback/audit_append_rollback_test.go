package audit_append_rollback_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/store"
)

func TestAuditAppendFailurePreservesCommittedSnapshot(t *testing.T) {
	dir := t.TempDir()
	repo, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := domain.NewBatch("batch", "持久化回滚", "operator", "reviewer", domain.DefaultTargetProfile(), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	first, err := repo.Commit(store.CommitRequest{Batch: batch, ExpectedVersion: 0, IdempotencyKey: "create", Command: "create", Actor: "operator", Result: batch, Details: map[string]string{"step": "create"}, Creating: true})
	if err != nil {
		t.Fatal(err)
	}
	auditPath := filepath.Join(dir, "audit.jsonl")
	backupPath := filepath.Join(dir, "audit.backup")
	if err := os.Rename(auditPath, backupPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(auditPath, 0700); err != nil {
		t.Fatal(err)
	}
	updated, err := domain.CloneBatch(first.Batch)
	if err != nil {
		t.Fatal(err)
	}
	updated.Title = "不应提交的更新"
	updated.Touch(time.Unix(2, 0))
	_, commitErr := repo.Commit(store.CommitRequest{Batch: updated, ExpectedVersion: first.Batch.Version, IdempotencyKey: "update", Command: "update", Actor: "operator", Result: updated, Details: map[string]string{"step": "update"}})
	if commitErr == nil {
		t.Fatal("审计目标不可写时提交应失败")
	}
	if err := os.Remove(auditPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backupPath, auditPath); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dir)
	if err != nil {
		t.Fatalf("失败提交后应能恢复此前快照: %v", err)
	}
	loaded, err := reopened.Get("batch")
	if err != nil {
		t.Fatalf("失败提交删除了此前已提交批次: %v", err)
	}
	if loaded.Version != first.Batch.Version || loaded.Title != first.Batch.Title {
		t.Fatalf("失败提交污染了持久化状态: version=%d title=%q", loaded.Version, loaded.Title)
	}
}
