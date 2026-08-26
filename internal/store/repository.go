package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"tape-preservation-gate/internal/domain"
	"time"
)

var ErrVersionConflict = errors.New("批次版本冲突")

type Repository struct {
	mu           sync.Mutex
	dir          string
	snapshotPath string
	auditPath    string
	state        Snapshot
}

func Open(dir string) (*Repository, error) {
	if dir == "" {
		return nil, fmt.Errorf("数据目录不能为空")
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("创建数据目录: %w", err)
	}
	r := &Repository{dir: dir, snapshotPath: filepath.Join(dir, "snapshot.json"), auditPath: filepath.Join(dir, "audit.jsonl")}
	r.state = Snapshot{SchemaVersion: SchemaVersion, Batches: map[string]*domain.DigitizationBatch{}, Idempotency: map[string]IdempotencyRecord{}, Audit: []AuditEvent{}}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Repository) List() ([]*domain.DigitizationBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*domain.DigitizationBatch, 0, len(r.state.Batches))
	for _, batch := range r.state.Batches {
		copy, err := domain.CloneBatch(batch)
		if err != nil {
			return nil, err
		}
		result = append(result, copy)
	}
	return result, nil
}

func (r *Repository) Get(id string) (*domain.DigitizationBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	batch, ok := r.state.Batches[id]
	if !ok {
		return nil, domain.NotFound("批次 %s 不存在", id)
	}
	return domain.CloneBatch(batch)
}

func (r *Repository) AuditForBatch(id string) []AuditEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := []AuditEvent{}
	for _, event := range r.state.Audit {
		if event.BatchID == id {
			result = append(result, event)
		}
	}
	return result
}

func (r *Repository) LastAuditDigest(id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.state.Audit) - 1; i >= 0; i-- {
		if r.state.Audit[i].BatchID == id {
			return r.state.Audit[i].Digest
		}
	}
	return ""
}

func (r *Repository) Replay(batchID, key, command string) (json.RawMessage, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.state.Idempotency[batchID+":"+key]
	if !ok {
		return nil, false, nil
	}
	if record.Command != command {
		return nil, false, domain.Invalid("同一 Idempotency-Key 不能用于不同命令")
	}
	return append(json.RawMessage(nil), record.Result...), true, nil
}

func (r *Repository) Close() error { return nil }

func nowUTC() time.Time { return time.Now().UTC() }

func marshalResult(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	return json.RawMessage(b), err
}
