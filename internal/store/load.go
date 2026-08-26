package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func (r *Repository) load() error {
	b, err := os.ReadFile(r.snapshotPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取快照: %w", err)
	}
	var state Snapshot
	if err := json.Unmarshal(b, &state); err != nil {
		return fmt.Errorf("快照 JSON 损坏: %w", err)
	}
	if state.SchemaVersion != SchemaVersion {
		return fmt.Errorf("不支持的快照 schemaVersion %d", state.SchemaVersion)
	}
	if state.Batches == nil || state.Idempotency == nil {
		return fmt.Errorf("快照缺少必要映射")
	}
	if err := validateAudit(state.Audit); err != nil {
		return fmt.Errorf("快照审计链损坏: %w", err)
	}
	for id, batch := range state.Batches {
		if batch == nil || batch.ID != id || batch.Version == 0 {
			return fmt.Errorf("批次 %s 快照结构无效", id)
		}
	}
	if err := r.validateAuditFile(state.Audit); err != nil {
		return err
	}
	r.state = state
	return nil
}

func (r *Repository) validateAuditFile(expected []AuditEvent) error {
	f, err := os.Open(r.auditPath)
	if errors.Is(err, os.ErrNotExist) && len(expected) == 0 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取审计日志: %w", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	actual := []AuditEvent{}
	for scanner.Scan() {
		var e AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return fmt.Errorf("审计日志 JSON 损坏: %w", err)
		}
		actual = append(actual, e)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(actual) != len(expected) {
		return fmt.Errorf("审计日志事件数与快照不一致")
	}
	for i := range actual {
		if actual[i].Digest != expected[i].Digest {
			return fmt.Errorf("审计日志第 %d 条与快照不一致", i+1)
		}
	}
	return validateAudit(actual)
}

func validateAudit(events []AuditEvent) error {
	previous := ""
	for i, event := range events {
		if event.Sequence != uint64(i+1) {
			return fmt.Errorf("事件序号不连续")
		}
		if event.PreviousDigest != previous {
			return fmt.Errorf("事件前序摘要不匹配")
		}
		digest, err := auditDigest(event)
		if err != nil {
			return err
		}
		if digest != event.Digest {
			return fmt.Errorf("事件摘要不匹配")
		}
		previous = event.Digest
	}
	return nil
}
