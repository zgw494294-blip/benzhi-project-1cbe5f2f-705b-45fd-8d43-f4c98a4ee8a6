package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

type ManifestEntry struct {
	CarrierID    string `json:"carrierID"`
	SegmentID    string `json:"segmentID"`
	CaptureRunID string `json:"captureRunID"`
	OutputFile   string `json:"outputFile"`
	Checksum     string `json:"checksumSHA256"`
	Attempt      int    `json:"attempt"`
}

func (b *DigitizationBatch) Manifest() []ManifestEntry {
	runs := b.LatestRuns()
	entries := make([]ManifestEntry, 0, len(runs))
	for _, r := range runs {
		entries = append(entries, ManifestEntry{CarrierID: r.CarrierID, SegmentID: r.SegmentID, CaptureRunID: r.ID, OutputFile: r.OutputFile, Checksum: r.ChecksumSHA256, Attempt: r.Attempt})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CarrierID != entries[j].CarrierID {
			return entries[i].CarrierID < entries[j].CarrierID
		}
		return entries[i].SegmentID < entries[j].SegmentID
	})
	return entries
}

// findingDigestEntry is the canonical, tamper-evident projection of a
// QualityFinding used to bind the certificate to the quality conclusions
// reached at sealing time. It captures the rule identity, evidence, severity
// and the resolution/acceptance conclusions so that any later mutation of the
// persisted findings (for example flipping a severity or rewriting a warning
// acceptance) changes the digest and is detected by VerifyCertificate.
type findingDigestEntry struct {
	ID                string                 `json:"id"`
	CaptureRunID      string                 `json:"captureRunID"`
	RuleCode          string                 `json:"ruleCode"`
	RuleVersion       string                 `json:"ruleVersion"`
	Severity          Severity               `json:"severity"`
	StartMillis       int64                  `json:"startMillis"`
	EndMillis         int64                  `json:"endMillis"`
	Evidence          map[string]string      `json:"evidence"`
	Status            FindingStatus          `json:"status"`
	Resolution        string                 `json:"resolution,omitempty"`
	ReplacementRunID  string                 `json:"replacementRunID,omitempty"`
	WarningAcceptance *warningAcceptanceView `json:"warningAcceptance,omitempty"`
}

// warningAcceptanceView is the canonical projection of WarningAcceptance. The
// EvidenceIdentity is excluded because it is itself a digest of the finding
// identity and would make the projection circular; the remaining fields fully
// describe the conclusion that was reached when the warning was accepted.
type warningAcceptanceView struct {
	ReasonCode  string `json:"reasonCode"`
	Description string `json:"description"`
	AcceptedBy  string `json:"acceptedBy"`
	AcceptedAt  string `json:"acceptedAt"`
}

// FindingsDigest returns a stable digest over the canonical projection of the
// batch's quality findings, so that the certificate can bind to and later
// verify the integrity of the quality conclusions in addition to the file
// manifest.
func (b *DigitizationBatch) FindingsDigest() (string, error) {
	entries := make([]findingDigestEntry, 0, len(b.Findings))
	for _, f := range b.Findings {
		var acceptance *warningAcceptanceView
		if f.WarningAcceptance != nil {
			acceptance = &warningAcceptanceView{ReasonCode: f.WarningAcceptance.ReasonCode, Description: f.WarningAcceptance.Description, AcceptedBy: f.WarningAcceptance.AcceptedBy, AcceptedAt: f.WarningAcceptance.AcceptedAt.UTC().Format(time.RFC3339Nano)}
		}
		entries = append(entries, findingDigestEntry{ID: f.ID, CaptureRunID: f.CaptureRunID, RuleCode: f.RuleCode, RuleVersion: f.RuleVersion, Severity: f.Severity, StartMillis: f.StartMillis, EndMillis: f.EndMillis, Evidence: f.Evidence, Status: f.Status, Resolution: f.Resolution, ReplacementRunID: f.ReplacementRunID, WarningAcceptance: acceptance})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CaptureRunID != entries[j].CaptureRunID {
			return entries[i].CaptureRunID < entries[j].CaptureRunID
		}
		if entries[i].RuleCode != entries[j].RuleCode {
			return entries[i].RuleCode < entries[j].RuleCode
		}
		if entries[i].StartMillis != entries[j].StartMillis {
			return entries[i].StartMillis < entries[j].StartMillis
		}
		return entries[i].ID < entries[j].ID
	})
	return StableDigest(entries)
}

func StableDigest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

type CertificatePayload struct {
	ID             string `json:"id"`
	BatchID        string `json:"batchID"`
	BatchVersion   uint64 `json:"batchVersion"`
	ManifestDigest string `json:"manifestDigest"`
	FindingsDigest string `json:"findingsDigest"`
	AuditDigest    string `json:"auditDigest"`
	RuleSetVersion string `json:"ruleSetVersion"`
	IssuedBy       string `json:"issuedBy"`
	IssuedAt       string `json:"issuedAt"`
}

func CertificateContent(c AcceptanceCertificate) CertificatePayload {
	return CertificatePayload{ID: c.ID, BatchID: c.BatchID, BatchVersion: c.BatchVersion, ManifestDigest: c.ManifestDigest, FindingsDigest: c.FindingsDigest, AuditDigest: c.AuditDigest, RuleSetVersion: c.RuleSetVersion, IssuedBy: c.IssuedBy, IssuedAt: c.IssuedAt.UTC().Format("2006-01-02T15:04:05Z")}
}
