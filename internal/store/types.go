package store

import (
	"encoding/json"
	"tape-preservation-gate/internal/domain"
	"time"
)

const SchemaVersion = 1

type Snapshot struct {
	SchemaVersion int                                  `json:"schemaVersion"`
	Batches       map[string]*domain.DigitizationBatch `json:"batches"`
	Idempotency   map[string]IdempotencyRecord         `json:"idempotency"`
	Audit         []AuditEvent                         `json:"audit"`
}

type IdempotencyRecord struct {
	BatchID   string          `json:"batchID"`
	Command   string          `json:"command"`
	Result    json.RawMessage `json:"result"`
	CreatedAt time.Time       `json:"createdAt"`
}

type AuditEvent struct {
	Sequence       uint64          `json:"sequence"`
	BatchID        string          `json:"batchID"`
	BatchVersion   uint64          `json:"batchVersion"`
	Command        string          `json:"command"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Actor          string          `json:"actor"`
	OccurredAt     time.Time       `json:"occurredAt"`
	Details        json.RawMessage `json:"details"`
	PreviousDigest string          `json:"previousDigest"`
	Digest         string          `json:"digest"`
}

type CommitRequest struct {
	Batch           *domain.DigitizationBatch
	ExpectedVersion uint64
	IdempotencyKey  string
	Command         string
	Actor           string
	Result          any
	Details         any
	Creating        bool
}

type CommitResult struct {
	Replay bool
	Result json.RawMessage
	Batch  *domain.DigitizationBatch
}
