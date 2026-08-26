package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
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
	AuditDigest    string `json:"auditDigest"`
	RuleSetVersion string `json:"ruleSetVersion"`
	IssuedBy       string `json:"issuedBy"`
	IssuedAt       string `json:"issuedAt"`
}

func CertificateContent(c AcceptanceCertificate) CertificatePayload {
	return CertificatePayload{ID: c.ID, BatchID: c.BatchID, BatchVersion: c.BatchVersion, ManifestDigest: c.ManifestDigest, AuditDigest: c.AuditDigest, RuleSetVersion: c.RuleSetVersion, IssuedBy: c.IssuedBy, IssuedAt: c.IssuedAt.UTC().Format("2006-01-02T15:04:05Z")}
}
