package store

import (
	"encoding/json"
	"fmt"
	"tape-preservation-gate/internal/domain"
)

func (r *Repository) Commit(req CommitRequest) (CommitResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if req.Batch == nil || req.IdempotencyKey == "" || req.Command == "" || req.Actor == "" {
		return CommitResult{}, domain.Invalid("提交缺少批次、幂等键、命令或操作者")
	}
	compoundKey := req.Batch.ID + ":" + req.IdempotencyKey
	if old, ok := r.state.Idempotency[compoundKey]; ok {
		if old.Command != req.Command {
			return CommitResult{}, domain.Invalid("同一 Idempotency-Key 不能用于不同命令")
		}
		batch, err := domain.CloneBatch(r.state.Batches[old.BatchID])
		if err != nil {
			return CommitResult{}, err
		}
		return CommitResult{Replay: true, Result: old.Result, Batch: batch}, nil
	}
	current, exists := r.state.Batches[req.Batch.ID]
	if req.Creating {
		if exists {
			return CommitResult{}, fmt.Errorf("%w: 批次已存在", ErrVersionConflict)
		}
		if req.ExpectedVersion != 0 {
			return CommitResult{}, fmt.Errorf("%w: 新建批次期望版本必须为 0", ErrVersionConflict)
		}
	} else {
		if !exists {
			return CommitResult{}, domain.NotFound("批次 %s 不存在", req.Batch.ID)
		}
		if current.Version != req.ExpectedVersion {
			return CommitResult{}, fmt.Errorf("%w: 当前版本 %d，提交版本 %d", ErrVersionConflict, current.Version, req.ExpectedVersion)
		}
		if req.Batch.Version != current.Version+1 {
			return CommitResult{}, domain.Invalid("提交后的批次版本必须递增 1")
		}
	}
	result, err := marshalResult(req.Result)
	if err != nil {
		return CommitResult{}, err
	}
	details, err := json.Marshal(req.Details)
	if err != nil {
		return CommitResult{}, err
	}
	event := AuditEvent{Sequence: uint64(len(r.state.Audit) + 1), BatchID: req.Batch.ID, BatchVersion: req.Batch.Version, Command: req.Command, IdempotencyKey: req.IdempotencyKey, Actor: req.Actor, OccurredAt: nowUTC(), Details: details, PreviousDigest: chainDigest(r.state.Audit)}
	event.Digest, err = auditDigest(event)
	if err != nil {
		return CommitResult{}, err
	}
	copy, err := domain.CloneBatch(req.Batch)
	if err != nil {
		return CommitResult{}, err
	}
	next := Snapshot{SchemaVersion: SchemaVersion, Batches: make(map[string]*domain.DigitizationBatch, len(r.state.Batches)+1), Idempotency: make(map[string]IdempotencyRecord, len(r.state.Idempotency)+1), Audit: append([]AuditEvent{}, r.state.Audit...)}
	for k, v := range r.state.Batches {
		cloned, e := domain.CloneBatch(v)
		if e != nil {
			return CommitResult{}, e
		}
		next.Batches[k] = cloned
	}
	for k, v := range r.state.Idempotency {
		next.Idempotency[k] = v
	}
	next.Batches[copy.ID] = copy
	next.Idempotency[compoundKey] = IdempotencyRecord{BatchID: copy.ID, Command: req.Command, Result: result, CreatedAt: nowUTC()}
	next.Audit = append(next.Audit, event)
	if err := atomicJSON(r.snapshotPath, next); err != nil {
		return CommitResult{}, fmt.Errorf("写入快照: %w", err)
	}
	if err := appendAudit(r.auditPath, event); err != nil {
		// 快照已被覆盖为本次提交状态，而审计追加失败。为使重启回到本次提交前的状态，
		// 用仍在内存中的提交前状态 r.state 重建快照与审计介质，丢弃追加可能留下的半成品。
		if rerr := atomicJSON(r.snapshotPath, r.state); rerr != nil {
			return CommitResult{}, fmt.Errorf("审计追加失败后回滚快照: %w (原审计错误: %v)", rerr, err)
		}
		if rerr := rewriteAudit(r.auditPath, r.state.Audit); rerr != nil {
			return CommitResult{}, fmt.Errorf("审计追加失败后回滚审计介质: %w (原审计错误: %v)", rerr, err)
		}
		return CommitResult{}, err
	}
	r.state = next
	return CommitResult{Result: result, Batch: copy}, nil
}
