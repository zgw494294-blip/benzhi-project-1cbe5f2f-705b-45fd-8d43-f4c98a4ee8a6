package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
	"time"
)

type Service struct {
	repo    *store.Repository
	quality *quality.Engine
	now     func() time.Time
}

func New(repo *store.Repository, engine *quality.Engine) *Service {
	return &Service{repo: repo, quality: engine, now: time.Now}
}

func randomID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(b), nil
}

func validateMeta(meta CommandMeta) error {
	if strings.TrimSpace(meta.IdempotencyKey) == "" {
		return domain.Invalid("缺少 Idempotency-Key")
	}
	if strings.TrimSpace(meta.Actor) == "" {
		return domain.Invalid("actor 不能为空")
	}
	return nil
}

func (s *Service) load(batchID, command string, meta CommandMeta) (*domain.DigitizationBatch, json.RawMessage, error) {
	if err := validateMeta(meta); err != nil {
		return nil, nil, err
	}
	if replay, ok, err := s.repo.Replay(batchID, meta.IdempotencyKey, command); err != nil || ok {
		return nil, replay, err
	}
	batch, err := s.repo.Get(batchID)
	if err != nil {
		return nil, nil, err
	}
	if batch.Version != meta.ExpectedVersion {
		return nil, nil, fmt.Errorf("%w: 当前版本 %d，提交版本 %d", store.ErrVersionConflict, batch.Version, meta.ExpectedVersion)
	}
	return batch, nil, nil
}

func replayBatch(raw json.RawMessage) (*domain.DigitizationBatch, error) {
	if raw == nil {
		return nil, nil
	}
	var batch domain.DigitizationBatch
	if err := json.Unmarshal(raw, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *Service) commit(batch *domain.DigitizationBatch, meta CommandMeta, command string, details any, creating bool) (*domain.DigitizationBatch, error) {
	result, err := s.repo.Commit(store.CommitRequest{Batch: batch, ExpectedVersion: meta.ExpectedVersion, IdempotencyKey: meta.IdempotencyKey, Command: command, Actor: meta.Actor, Result: batch, Details: details, Creating: creating})
	if err != nil {
		return nil, err
	}
	if result.Replay {
		return replayBatch(result.Result)
	}
	return result.Batch, nil
}

func IsConflict(err error) bool { return errors.Is(err, store.ErrVersionConflict) }

func (s *Service) ListBatches() ([]*domain.DigitizationBatch, error) {
	items, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *Service) GetBatch(id string) (*domain.DigitizationBatch, error) { return s.repo.Get(id) }
