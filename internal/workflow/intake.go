package workflow

import (
	"strings"
	"tape-preservation-gate/internal/domain"
)

func (s *Service) CreateBatch(cmd CreateBatchCommand) (*domain.DigitizationBatch, error) {
	if err := validateMeta(cmd.Meta); err != nil {
		return nil, err
	}
	if cmd.ID == "" {
		id, err := randomID("batch")
		if err != nil {
			return nil, err
		}
		cmd.ID = id
	}
	if replay, ok, err := s.repo.Replay(cmd.ID, cmd.Meta.IdempotencyKey, "create_batch"); err != nil || ok {
		if err != nil {
			return nil, err
		}
		return replayBatch(replay)
	}
	if cmd.TargetProfile.Codec == "" {
		cmd.TargetProfile = domain.DefaultTargetProfile()
	}
	batch, err := domain.NewBatch(cmd.ID, cmd.Title, cmd.Operator, cmd.Reviewer, cmd.TargetProfile, s.now())
	if err != nil {
		return nil, err
	}
	return s.commit(batch, cmd.Meta, "create_batch", map[string]any{"title": cmd.Title, "operator": cmd.Operator, "reviewer": cmd.Reviewer}, true)
}

func (s *Service) AddCarrier(batchID string, cmd AddCarrierCommand) (*domain.DigitizationBatch, error) {
	batch, replay, err := s.load(batchID, "add_carrier", cmd.Meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	if cmd.Carrier.ID == "" {
		cmd.Carrier.ID, err = randomID("carrier")
		if err != nil {
			return nil, err
		}
	}
	for i := range cmd.Carrier.Segments {
		if cmd.Carrier.Segments[i].ID == "" {
			cmd.Carrier.Segments[i].ID, err = randomID("segment")
			if err != nil {
				return nil, err
			}
		}
	}
	if err := batch.AddCarrier(cmd.Carrier, s.now()); err != nil {
		return nil, err
	}
	carrier, _ := batch.Carrier(cmd.Carrier.ID)
	ranges := make([]map[string]any, 0, len(carrier.Segments))
	for _, segment := range carrier.Segments {
		ranges = append(ranges, map[string]any{"segmentID": segment.ID, "title": segment.Title, "startMillis": segment.StartMillis, "endMillis": segment.StartMillis + segment.DurationMillis, "durationMillis": segment.DurationMillis})
	}
	return s.commit(batch, cmd.Meta, "add_carrier", map[string]any{"carrierID": cmd.Carrier.ID, "archiveCode": cmd.Carrier.ArchiveCode, "durationMillis": carrier.DurationMillis, "segments": ranges}, false)
}

func (s *Service) InspectCarrier(batchID string, cmd InspectCarrierCommand) (*domain.DigitizationBatch, error) {
	batch, replay, err := s.load(batchID, "inspect_carrier", cmd.Meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	action := strings.TrimSpace(cmd.Action)
	if action == "" || action == "inspect" || action == "reinspect" {
		if cmd.Inspection.ID == "" {
			cmd.Inspection.ID, err = randomID("inspection")
			if err != nil {
				return nil, err
			}
		}
		if err := batch.InspectCarrier(cmd.CarrierID, cmd.Inspection, s.now()); err != nil {
			return nil, err
		}
		carrier, _ := batch.Carrier(cmd.CarrierID)
		return s.commit(batch, cmd.Meta, "inspect_carrier", map[string]any{"action": action, "carrierID": cmd.CarrierID, "inspectionID": cmd.Inspection.ID, "riskCodes": carrier.Inspection.RiskCodes, "eligibility": carrier.CaptureEligibility}, false)
	}
	if action != "treat" {
		return nil, domain.Invalid("盘前检查动作 %s 无效", action)
	}
	if cmd.Treatment.ID == "" {
		cmd.Treatment.ID, err = randomID("treatment")
		if err != nil {
			return nil, err
		}
	}
	if err := batch.TreatCarrierRisks(cmd.CarrierID, cmd.Treatment, s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, cmd.Meta, "inspect_carrier", map[string]any{"action": action, "carrierID": cmd.CarrierID, "treatmentID": cmd.Treatment.ID, "riskCodes": cmd.Treatment.RiskCodes}, false)
}

func (s *Service) FreezePlan(batchID string, meta CommandMeta) (*domain.DigitizationBatch, error) {
	batch, replay, err := s.load(batchID, "freeze_plan", meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	if err := batch.FreezePlan(s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, meta, "freeze_plan", map[string]any{"plannedSegments": batch.PlannedSegmentCount()}, false)
}
