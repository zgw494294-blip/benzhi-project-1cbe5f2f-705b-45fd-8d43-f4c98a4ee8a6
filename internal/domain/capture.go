package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var checksumPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func (b *DigitizationBatch) AddCaptureRun(run CaptureRun, now time.Time) error {
	if err := b.addCaptureRun(run, now); err != nil {
		return err
	}
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) addCaptureRun(run CaptureRun, now time.Time) error {
	if err := b.EnsureMutable(); err != nil {
		return err
	}
	if b.State != StatePlanFrozen && b.State != StateQualityReady && b.State != StateRemediation {
		return StateError("当前状态不能登记采集")
	}
	if strings.TrimSpace(run.ID) == "" || strings.TrimSpace(run.OutputFile) == "" || len(run.EquipmentChain) == 0 {
		return Invalid("采集轮次、文件和设备链不能为空")
	}
	for _, equipment := range run.EquipmentChain {
		if strings.TrimSpace(equipment) == "" {
			return Invalid("设备链不能包含空设备")
		}
	}
	if !checksumPattern.MatchString(run.ChecksumSHA256) {
		return Invalid("SHA-256 校验和必须为 64 位十六进制")
	}
	if run.Measurements.SilenceMillis < 0 || run.Measurements.DropoutMillis < 0 || run.Measurements.MeasuredDuration <= 0 {
		return Invalid("技术测量值不完整：静音、掉音不能为负且实测时长必须为正数")
	}
	c, err := b.Carrier(run.CarrierID)
	if err != nil {
		return err
	}
	segmentOK := false
	for _, s := range c.Segments {
		if s.ID == run.SegmentID {
			segmentOK = true
		}
	}
	if !segmentOK {
		return Invalid("节目段不属于指定载体")
	}
	maxAttempt := 0
	for _, existing := range b.CaptureRuns {
		if existing.ID == run.ID || existing.OutputFile == run.OutputFile {
			return Invalid("采集轮次 ID 或输出文件重复")
		}
		if existing.CarrierID == run.CarrierID && existing.SegmentID == run.SegmentID && existing.Attempt > maxAttempt {
			maxAttempt = existing.Attempt
		}
	}
	if run.Attempt != maxAttempt+1 {
		return Invalid("采集尝试次序应为 %d", maxAttempt+1)
	}
	if run.Attempt > 1 {
		if run.SupersedesRunID == "" {
			return Invalid("重采必须关联被替代轮次")
		}
		old, err := b.Capture(run.SupersedesRunID)
		if err != nil {
			return err
		}
		if old.CarrierID != run.CarrierID || old.SegmentID != run.SegmentID {
			return Invalid("替代轮次必须属于相同载体和节目段")
		}
		if old.Attempt != run.Attempt-1 {
			return Invalid("重采必须直接关联同目标的第 %d 次采集", run.Attempt-1)
		}
	}
	run.BatchID = b.ID
	run.CapturedAt = now.UTC()
	run.ChecksumSHA256 = strings.ToLower(run.ChecksumSHA256)
	b.CaptureRuns = append(b.CaptureRuns, run)
	b.QualityRunAt = nil
	return nil
}

// AddCaptureRuns 在聚合副本上按请求顺序校验，只有全部行通过才把结果写回，
// 因而同一请求内的 attempt 和 supersedesRunID 也参与连续性判断。
func (b *DigitizationBatch) AddCaptureRuns(runs []CaptureRun, now time.Time) error {
	if len(runs) == 0 {
		return Invalid("批量采集至少需要一行")
	}
	candidate, err := CloneBatch(b)
	if err != nil {
		return err
	}
	problems := []string{}
	for i, run := range runs {
		if err := candidate.addCaptureRun(run, now); err != nil {
			target := strings.TrimSpace(run.CarrierID) + "/" + strings.TrimSpace(run.SegmentID)
			problems = append(problems, fmt.Sprintf("第 %d 行（%s）：%v", i+1, target, err))
		}
	}
	if len(problems) > 0 {
		return Invalid("批量采集校验失败：%s", strings.Join(problems, "；"))
	}
	*b = *candidate
	b.Touch(now)
	return nil
}

func (b *DigitizationBatch) LatestRuns() []CaptureRun {
	latest := map[string]CaptureRun{}
	for _, run := range b.CaptureRuns {
		key := run.CarrierID + "/" + run.SegmentID
		if old, ok := latest[key]; !ok || run.Attempt > old.Attempt {
			latest[key] = run
		}
	}
	result := make([]CaptureRun, 0, len(latest))
	for _, run := range latest {
		result = append(result, run)
	}
	return result
}

func (b *DigitizationBatch) PlannedSegmentCount() int {
	n := 0
	for _, c := range b.Carriers {
		n += len(c.Segments)
	}
	return n
}
