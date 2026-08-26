package quality

import "tape-preservation-gate/internal/domain"

const RuleSetVersion = "audio-preservation-rules/1.0.0"

type Rule interface {
	Code() string
	Evaluate(domain.TargetProfile, domain.ProgramSegment, domain.CaptureRun) *domain.QualityFinding
}

type Engine struct{ rules []Rule }

func NewEngine() *Engine {
	return &Engine{rules: []Rule{checksumRule{}, peakRule{}, silenceRule{}, dropoutRule{}, timebaseRule{}, durationRule{}}}
}

func finding(code string, severity domain.Severity, run domain.CaptureRun, start, end int64, evidence map[string]string) *domain.QualityFinding {
	key := struct {
		Run, Code, RuleVersion string
		Start, End             int64
		Evidence               map[string]string
	}{run.ID, code, RuleSetVersion, start, end, evidence}
	digest, _ := domain.StableDigest(key)
	return &domain.QualityFinding{ID: "finding-" + digest[:16], CaptureRunID: run.ID, RuleCode: code, RuleVersion: RuleSetVersion, Severity: severity, StartMillis: start, EndMillis: end, Evidence: evidence, Status: domain.FindingOpen}
}
