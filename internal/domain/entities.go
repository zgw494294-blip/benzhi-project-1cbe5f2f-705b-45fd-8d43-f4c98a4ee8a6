package domain

import "time"

type DigitizationBatch struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	State          BatchState             `json:"state"`
	TargetProfile  TargetProfile          `json:"targetProfile"`
	Operator       string                 `json:"operator"`
	Reviewer       string                 `json:"reviewer"`
	Version        uint64                 `json:"version"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
	PlanFrozenAt   *time.Time             `json:"planFrozenAt,omitempty"`
	SubmissionNo   int                    `json:"submissionNo"`
	Carriers       []TapeCarrier          `json:"carriers"`
	CaptureRuns    []CaptureRun           `json:"captureRuns"`
	Findings       []QualityFinding       `json:"findings"`
	Reviews        []ReviewDecision       `json:"reviews"`
	Remediations   []RemediationItem      `json:"remediations"`
	Certificate    *AcceptanceCertificate `json:"certificate,omitempty"`
	QualityRunAt   *time.Time             `json:"qualityRunAt,omitempty"`
	QualityVersion string                 `json:"qualityVersion,omitempty"`
}

type TapeCarrier struct {
	ID                 string                 `json:"id"`
	BatchID            string                 `json:"batchID"`
	ArchiveCode        string                 `json:"archiveCode"`
	Format             string                 `json:"format"`
	DurationMillis     int64                  `json:"durationMillis"`
	Segments           []ProgramSegment       `json:"segments"`
	Inspection         CarrierInspection      `json:"inspection"`
	InspectionHistory  []CarrierInspection    `json:"inspectionHistory"`
	RiskTreatments     []CarrierRiskTreatment `json:"riskTreatments"`
	CaptureEligibility Eligibility            `json:"captureEligibility"`
	Notes              string                 `json:"notes"`
}

type CaptureRun struct {
	ID              string             `json:"id"`
	BatchID         string             `json:"batchID"`
	CarrierID       string             `json:"carrierID"`
	SegmentID       string             `json:"segmentID"`
	Attempt         int                `json:"attempt"`
	EquipmentChain  []string           `json:"equipmentChain"`
	OutputFile      string             `json:"outputFile"`
	ChecksumSHA256  string             `json:"checksumSHA256"`
	Measurements    SignalMeasurements `json:"measurements"`
	CapturedAt      time.Time          `json:"capturedAt"`
	SupersedesRunID string             `json:"supersedesRunID,omitempty"`
}

type QualityFinding struct {
	ID                string             `json:"id"`
	BatchID           string             `json:"batchID"`
	CaptureRunID      string             `json:"captureRunID"`
	RuleCode          string             `json:"ruleCode"`
	RuleVersion       string             `json:"ruleVersion"`
	Severity          Severity           `json:"severity"`
	StartMillis       int64              `json:"startMillis"`
	EndMillis         int64              `json:"endMillis"`
	Evidence          map[string]string  `json:"evidence"`
	Status            FindingStatus      `json:"status"`
	Resolution        string             `json:"resolution,omitempty"`
	ReplacementRunID  string             `json:"replacementRunID,omitempty"`
	WarningAcceptance *WarningAcceptance `json:"warningAcceptance,omitempty"`
}

type ReviewDecision struct {
	ID           string         `json:"id"`
	BatchID      string         `json:"batchID"`
	SubmissionNo int            `json:"submissionNo"`
	Decision     Decision       `json:"decision"`
	ReasonCodes  []string       `json:"reasonCodes"`
	Reasons      []ReviewReason `json:"reasons,omitempty"`
	Comment      string         `json:"comment"`
	Reviewer     string         `json:"reviewer"`
	DecidedAt    time.Time      `json:"decidedAt"`
}

type RemediationItem struct {
	ID                 string                     `json:"id"`
	BatchID            string                     `json:"batchID"`
	SubmissionNo       int                        `json:"submissionNo"`
	ReasonCode         string                     `json:"reasonCode"`
	ProblemDescription string                     `json:"problemDescription"`
	Assignee           string                     `json:"assignee"`
	CreatedAt          time.Time                  `json:"createdAt"`
	Status             RemediationStatus          `json:"status"`
	Resolution         string                     `json:"resolution,omitempty"`
	Evidence           *BusinessEvidenceReference `json:"evidence,omitempty"`
	ClosedBy           string                     `json:"closedBy,omitempty"`
	ClosedAt           *time.Time                 `json:"closedAt,omitempty"`
}

type AcceptanceCertificate struct {
	ID                string    `json:"id"`
	BatchID           string    `json:"batchID"`
	BatchVersion      uint64    `json:"batchVersion"`
	ManifestDigest    string    `json:"manifestDigest"`
	AuditDigest       string    `json:"auditDigest"`
	RuleSetVersion    string    `json:"ruleSetVersion"`
	IssuedBy          string    `json:"issuedBy"`
	IssuedAt          time.Time `json:"issuedAt"`
	CertificateDigest string    `json:"certificateDigest"`
}
