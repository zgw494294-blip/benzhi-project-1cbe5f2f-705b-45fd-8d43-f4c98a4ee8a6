package workflow

import "tape-preservation-gate/internal/domain"

type CommandMeta struct {
	ExpectedVersion uint64 `json:"expectedVersion"`
	IdempotencyKey  string `json:"-"`
	Actor           string `json:"actor"`
}

type CreateBatchCommand struct {
	ID            string               `json:"id"`
	Title         string               `json:"title"`
	Operator      string               `json:"operator"`
	Reviewer      string               `json:"reviewer"`
	TargetProfile domain.TargetProfile `json:"targetProfile"`
	Meta          CommandMeta          `json:"-"`
}

type AddCarrierCommand struct {
	Meta    CommandMeta        `json:"-"`
	Carrier domain.TapeCarrier `json:"carrier"`
}

type InspectCarrierCommand struct {
	Meta       CommandMeta                 `json:"-"`
	Action     string                      `json:"action"`
	CarrierID  string                      `json:"carrierID"`
	Inspection domain.CarrierInspection    `json:"inspection"`
	Treatment  domain.CarrierRiskTreatment `json:"treatment"`
}

type CaptureCommand struct {
	Meta CommandMeta         `json:"-"`
	Run  domain.CaptureRun   `json:"run"`
	Runs []domain.CaptureRun `json:"runs"`
}

type ResolveFindingCommand struct {
	Meta             CommandMeta `json:"-"`
	FindingID        string      `json:"findingID"`
	Resolution       string      `json:"resolution"`
	ReplacementRunID string      `json:"replacementRunID"`
	Action           string      `json:"action"`
	ReasonCode       string      `json:"reasonCode"`
	Description      string      `json:"description"`
}

type ReviewCommand struct {
	Meta        CommandMeta           `json:"-"`
	Decision    domain.Decision       `json:"decision"`
	ReasonCodes []string              `json:"reasonCodes"`
	Reasons     []domain.ReviewReason `json:"reasons"`
	Comment     string                `json:"comment"`
}

type SubmitReviewCommand struct {
	Meta         CommandMeta                    `json:"-"`
	Remediations []domain.RemediationResolution `json:"remediations"`
}

type CertificateVerification struct {
	Valid             bool   `json:"valid"`
	CertificateDigest string `json:"certificateDigest"`
	ManifestDigest    string `json:"manifestDigest"`
	AuditDigest       string `json:"auditDigest"`
	Message           string `json:"message"`
}
