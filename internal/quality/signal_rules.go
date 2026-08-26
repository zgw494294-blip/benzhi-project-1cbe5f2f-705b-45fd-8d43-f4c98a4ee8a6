package quality

import (
	"fmt"
	"math"
	"regexp"
	"tape-preservation-gate/internal/domain"
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type checksumRule struct{}

func (checksumRule) Code() string { return "CHECKSUM_FORMAT" }
func (r checksumRule) Evaluate(_ domain.TargetProfile, s domain.ProgramSegment, run domain.CaptureRun) *domain.QualityFinding {
	if sha256Pattern.MatchString(run.ChecksumSHA256) {
		return nil
	}
	return finding(r.Code(), domain.SeverityBlocking, run, 0, s.DurationMillis, map[string]string{"actual": run.ChecksumSHA256, "expected": "64 位小写十六进制 SHA-256"})
}

type peakRule struct{}

func (peakRule) Code() string { return "PEAK_OVER_LIMIT" }
func (r peakRule) Evaluate(p domain.TargetProfile, s domain.ProgramSegment, run domain.CaptureRun) *domain.QualityFinding {
	if run.Measurements.PeakDBFS <= p.PeakLimitDBFS {
		return nil
	}
	return finding(r.Code(), domain.SeverityBlocking, run, 0, s.DurationMillis, map[string]string{"actualDBFS": fmt.Sprintf("%.2f", run.Measurements.PeakDBFS), "limitDBFS": fmt.Sprintf("%.2f", p.PeakLimitDBFS)})
}

type silenceRule struct{}

func (silenceRule) Code() string { return "LONG_SILENCE" }
func (r silenceRule) Evaluate(p domain.TargetProfile, s domain.ProgramSegment, run domain.CaptureRun) *domain.QualityFinding {
	if run.Measurements.SilenceMillis <= p.MaxSilenceMillis {
		return nil
	}
	end := run.Measurements.SilenceMillis
	if end > s.DurationMillis {
		end = s.DurationMillis
	}
	return finding(r.Code(), domain.SeverityWarning, run, 0, end, map[string]string{"actualMillis": fmt.Sprint(run.Measurements.SilenceMillis), "limitMillis": fmt.Sprint(p.MaxSilenceMillis)})
}

type dropoutRule struct{}

func (dropoutRule) Code() string { return "DROPOUT" }
func (r dropoutRule) Evaluate(p domain.TargetProfile, s domain.ProgramSegment, run domain.CaptureRun) *domain.QualityFinding {
	if run.Measurements.DropoutMillis <= p.MaxDropoutMillis {
		return nil
	}
	end := run.Measurements.DropoutMillis
	if end > s.DurationMillis {
		end = s.DurationMillis
	}
	return finding(r.Code(), domain.SeverityBlocking, run, 0, end, map[string]string{"actualMillis": fmt.Sprint(run.Measurements.DropoutMillis), "limitMillis": fmt.Sprint(p.MaxDropoutMillis)})
}

type timebaseRule struct{}

func (timebaseRule) Code() string { return "TIMEBASE_DRIFT" }
func (r timebaseRule) Evaluate(p domain.TargetProfile, s domain.ProgramSegment, run domain.CaptureRun) *domain.QualityFinding {
	actual := math.Abs(run.Measurements.TimebasePPM)
	if actual <= p.MaxTimebasePPM {
		return nil
	}
	return finding(r.Code(), domain.SeverityBlocking, run, 0, s.DurationMillis, map[string]string{"actualPPM": fmt.Sprintf("%.2f", actual), "limitPPM": fmt.Sprintf("%.2f", p.MaxTimebasePPM)})
}

type durationRule struct{}

func (durationRule) Code() string { return "DURATION_MISMATCH" }
func (r durationRule) Evaluate(p domain.TargetProfile, s domain.ProgramSegment, run domain.CaptureRun) *domain.QualityFinding {
	delta := run.Measurements.MeasuredDuration - s.DurationMillis
	if math.Abs(float64(delta)) <= float64(p.DurationTolerance) {
		return nil
	}
	return finding(r.Code(), domain.SeverityBlocking, run, 0, s.DurationMillis, map[string]string{"actualMillis": fmt.Sprint(run.Measurements.MeasuredDuration), "plannedMillis": fmt.Sprint(s.DurationMillis), "deltaMillis": fmt.Sprint(delta), "toleranceMillis": fmt.Sprint(p.DurationTolerance)})
}
