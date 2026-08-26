package domain

import "time"

type BatchState string

const (
	StateDraft        BatchState = "draft"
	StatePlanFrozen   BatchState = "plan_frozen"
	StateQualityReady BatchState = "quality_ready"
	StateInReview     BatchState = "in_review"
	StateRemediation  BatchState = "remediation"
	StateApproved     BatchState = "approved"
	StateSealed       BatchState = "sealed"
)

type Eligibility string

const (
	EligibilityPending Eligibility = "pending"
	EligibilityAllowed Eligibility = "allowed"
	EligibilityBlocked Eligibility = "blocked"
)

type Severity string

const (
	SeverityBlocking Severity = "blocking"
	SeverityWarning  Severity = "warning"
)

type FindingStatus string

const (
	FindingOpen     FindingStatus = "open"
	FindingAccepted FindingStatus = "accepted"
	FindingResolved FindingStatus = "resolved"
	FindingClosed   FindingStatus = "closed"
)

type Decision string

const (
	DecisionReturned Decision = "returned"
	DecisionApproved Decision = "approved"
)

type TargetProfile struct {
	Codec             string  `json:"codec"`
	SampleRate        int     `json:"sampleRate"`
	BitDepth          int     `json:"bitDepth"`
	Channels          int     `json:"channels"`
	PeakLimitDBFS     float64 `json:"peakLimitDBFS"`
	MaxSilenceMillis  int64   `json:"maxSilenceMillis"`
	MaxDropoutMillis  int64   `json:"maxDropoutMillis"`
	MaxTimebasePPM    float64 `json:"maxTimebasePPM"`
	DurationTolerance int64   `json:"durationToleranceMillis"`
}

func DefaultTargetProfile() TargetProfile {
	return TargetProfile{Codec: "BWF/PCM", SampleRate: 96000, BitDepth: 24, Channels: 2, PeakLimitDBFS: -1, MaxSilenceMillis: 5000, MaxDropoutMillis: 100, MaxTimebasePPM: 50, DurationTolerance: 1000}
}

type ProgramSegment struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	StartMillis    int64  `json:"startMillis"`
	DurationMillis int64  `json:"durationMillis"`
}

type CarrierInspection struct {
	ID           string      `json:"id"`
	Sequence     int         `json:"sequence"`
	AppearanceOK bool        `json:"appearanceOK"`
	Mold         bool        `json:"mold"`
	Sticky       bool        `json:"sticky"`
	HubOK        bool        `json:"hubOK"`
	LeaderOK     bool        `json:"leaderOK"`
	CheckedBy    string      `json:"checkedBy"`
	CheckedAt    time.Time   `json:"checkedAt"`
	Comment      string      `json:"comment"`
	RiskCodes    []string    `json:"riskCodes"`
	Eligibility  Eligibility `json:"eligibility"`
}

type CarrierRiskTreatment struct {
	ID           string    `json:"id"`
	InspectionID string    `json:"inspectionID"`
	RiskCodes    []string  `json:"riskCodes"`
	Method       string    `json:"method"`
	Description  string    `json:"description"`
	ExecutedBy   string    `json:"executedBy"`
	ExecutedAt   time.Time `json:"executedAt"`
}

type WarningAcceptance struct {
	ReasonCode       string    `json:"reasonCode"`
	Description      string    `json:"description"`
	AcceptedBy       string    `json:"acceptedBy"`
	AcceptedAt       time.Time `json:"acceptedAt"`
	EvidenceIdentity string    `json:"evidenceIdentity"`
}

type ReviewReason struct {
	ReasonCode         string `json:"reasonCode"`
	ProblemDescription string `json:"problemDescription"`
	Assignee           string `json:"assignee"`
}

type RemediationStatus string

const (
	RemediationOpen   RemediationStatus = "open"
	RemediationClosed RemediationStatus = "closed"
)

type BusinessEvidenceReference struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	BatchID      string `json:"batchID"`
	SubmissionNo int    `json:"submissionNo"`
}

type RemediationResolution struct {
	ItemID      string                    `json:"itemID"`
	Description string                    `json:"description"`
	Evidence    BusinessEvidenceReference `json:"evidence"`
}

type SignalMeasurements struct {
	PeakDBFS         float64 `json:"peakDBFS"`
	SilenceMillis    int64   `json:"silenceMillis"`
	DropoutMillis    int64   `json:"dropoutMillis"`
	TimebasePPM      float64 `json:"timebasePPM"`
	MeasuredDuration int64   `json:"measuredDurationMillis"`
}
