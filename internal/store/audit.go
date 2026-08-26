package store

import (
	"encoding/json"
	"tape-preservation-gate/internal/domain"
)

type digestEvent struct {
	Sequence       uint64 `json:"sequence"`
	BatchID        string `json:"batchID"`
	BatchVersion   uint64 `json:"batchVersion"`
	Command        string `json:"command"`
	IdempotencyKey string `json:"idempotencyKey"`
	Actor          string `json:"actor"`
	OccurredAt     string `json:"occurredAt"`
	Details        any    `json:"details"`
	PreviousDigest string `json:"previousDigest"`
}

func auditDigest(e AuditEvent) (string, error) {
	var details any
	if err := json.Unmarshal(e.Details, &details); err != nil {
		return "", err
	}
	return domain.StableDigest(digestEvent{Sequence: e.Sequence, BatchID: e.BatchID, BatchVersion: e.BatchVersion, Command: e.Command, IdempotencyKey: e.IdempotencyKey, Actor: e.Actor, OccurredAt: e.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000000Z"), Details: details, PreviousDigest: e.PreviousDigest})
}

func chainDigest(events []AuditEvent) string {
	if len(events) == 0 {
		return ""
	}
	return events[len(events)-1].Digest
}
