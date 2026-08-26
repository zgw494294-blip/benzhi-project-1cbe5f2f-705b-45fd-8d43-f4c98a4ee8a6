package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"tape-preservation-gate/internal/domain"
	"testing"
	"time"
)

func TestCommitReplayAndRestart(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := domain.NewBatch("b1", "持久化测试", "op", "reviewer", domain.DefaultTargetProfile(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	req := CommitRequest{Batch: b, ExpectedVersion: 0, IdempotencyKey: "key-1", Command: "create", Actor: "op", Result: b, Details: map[string]string{"title": b.Title}, Creating: true}
	first, err := repo.Commit(req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Commit(req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replay || !second.Replay {
		t.Fatalf("幂等重放标记错误")
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get("b1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != b.Title || len(reopened.AuditForBatch("b1")) != 1 {
		t.Fatalf("恢复内容不完整")
	}
}

func TestRejectsBrokenAuditChain(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := domain.NewBatch("b1", "损坏测试", "op", "reviewer", domain.DefaultTargetProfile(), time.Now())
	_, err = repo.Commit(CommitRequest{Batch: b, ExpectedVersion: 0, IdempotencyKey: "k", Command: "create", Actor: "op", Result: b, Details: map[string]any{}, Creating: true})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "snapshot.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatal(err)
	}
	snap.Audit[0].Digest = "broken"
	data, err = json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("损坏摘要链应被拒绝")
	}
}
