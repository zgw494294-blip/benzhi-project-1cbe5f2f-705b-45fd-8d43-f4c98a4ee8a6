package workflow

import (
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/quality"
)

func (s *Service) IssueCertificate(batchID string, meta CommandMeta) (*domain.DigitizationBatch, error) {
	batch, replay, err := s.load(batchID, "issue_certificate", meta)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return replayBatch(replay)
	}
	manifestDigest, err := domain.StableDigest(batch.Manifest())
	if err != nil {
		return nil, err
	}
	findingsDigest, err := batch.FindingsDigest()
	if err != nil {
		return nil, err
	}
	auditDigest := s.repo.LastAuditDigest(batch.ID)
	id, err := randomID("certificate")
	if err != nil {
		return nil, err
	}
	c := domain.AcceptanceCertificate{ID: id, BatchID: batch.ID, BatchVersion: batch.Version + 1, ManifestDigest: manifestDigest, FindingsDigest: findingsDigest, AuditDigest: auditDigest, RuleSetVersion: quality.RuleSetVersion, IssuedBy: meta.Actor, IssuedAt: s.now().UTC()}
	c.CertificateDigest, err = domain.StableDigest(domain.CertificateContent(c))
	if err != nil {
		return nil, err
	}
	if err := batch.Seal(c, s.now()); err != nil {
		return nil, err
	}
	return s.commit(batch, meta, "issue_certificate", c, false)
}

func (s *Service) VerifyCertificate(batchID string) (CertificateVerification, error) {
	batch, err := s.repo.Get(batchID)
	if err != nil {
		return CertificateVerification{}, err
	}
	if batch.Certificate == nil {
		return CertificateVerification{Valid: false, Message: "批次尚未签发验收凭据"}, nil
	}
	c := *batch.Certificate
	manifestDigest, err := domain.StableDigest(batch.Manifest())
	if err != nil {
		return CertificateVerification{}, err
	}
	findingsDigest, err := batch.FindingsDigest()
	if err != nil {
		return CertificateVerification{}, err
	}
	digest, err := domain.StableDigest(domain.CertificateContent(c))
	if err != nil {
		return CertificateVerification{}, err
	}
	valid := digest == c.CertificateDigest && manifestDigest == c.ManifestDigest && findingsDigest == c.FindingsDigest && c.BatchID == batch.ID && c.BatchVersion == batch.Version
	message := "凭据内容、文件清单与摘要一致"
	if !valid {
		message = "凭据或保存文件清单摘要不一致"
	}
	return CertificateVerification{Valid: valid, CertificateDigest: digest, ManifestDigest: manifestDigest, AuditDigest: c.AuditDigest, Message: message}, nil
}
