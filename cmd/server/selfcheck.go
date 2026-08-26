package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"tape-preservation-gate/internal/domain"
	"time"
)

type selfcheckClient struct {
	base   string
	http   *http.Client
	serial int
}

func runSelfcheck(server *http.Server, listener net.Listener) error {
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	host := listener.Addr().String()
	client := &selfcheckClient{base: "http://" + host, http: &http.Client{Timeout: 5 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := client.run(ctx)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		if err == nil {
			err = fmt.Errorf("HTTP 服务未在时限内退出")
		}
	}
	if err != nil {
		return fmt.Errorf("自检失败: %w", err)
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	fmt.Println("selfcheck passed: 建批、检查、计划冻结、采集、质量检测、复核、封存和凭据核验均通过")
	return nil
}

func (c *selfcheckClient) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	c.serial++
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("selfcheck-%02d", c.serial))
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s 返回 %d: %s", path, resp.StatusCode, data)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}

func (c *selfcheckClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s 返回 %d: %s", path, resp.StatusCode, data)
	}
	return json.Unmarshal(data, out)
}

func (c *selfcheckClient) getText(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s 返回 %d", path, resp.StatusCode)
	}
	return string(data), nil
}

func (c *selfcheckClient) run(ctx context.Context) error {
	var health map[string]any
	if err := c.get(ctx, "/healthz", &health); err != nil {
		return err
	}
	page, err := c.getText(ctx, "/app")
	if err != nil {
		return err
	}
	if !bytes.Contains([]byte(page), []byte("磁带数字化质量验收工作台")) {
		return fmt.Errorf("工作台页面缺少预期内容")
	}
	create := map[string]any{"id": "selfcheck-batch", "title": "自检磁带验收批次", "operator": "operator-a", "reviewer": "reviewer-b", "actor": "operator-a", "targetProfile": domain.DefaultTargetProfile()}
	var batch domain.DigitizationBatch
	if err := c.post(ctx, "/api/v1/batches", create, &batch); err != nil {
		return err
	}
	carrier := domain.TapeCarrier{ID: "carrier-a", ArchiveCode: "SC-TAPE-001", Format: "1/4 inch open reel", DurationMillis: 60000, Segments: []domain.ProgramSegment{{ID: "segment-b", Title: "自检节目二", StartMillis: 30000, DurationMillis: 20000}, {ID: "segment-a", Title: "自检节目一", StartMillis: 0, DurationMillis: 20000}}}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/carriers", map[string]any{"expectedVersion": batch.Version, "actor": "operator-a", "carrier": carrier}, &batch); err != nil {
		return err
	}
	inspection := domain.CarrierInspection{AppearanceOK: true, HubOK: true, LeaderOK: true, CheckedBy: "operator-a", Comment: "自检安全载体"}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/carrier-inspections", map[string]any{"expectedVersion": batch.Version, "actor": "operator-a", "carrierID": "carrier-a", "inspection": inspection}, &batch); err != nil {
		return err
	}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/plan-freeze", map[string]any{"expectedVersion": batch.Version, "actor": "operator-a"}, &batch); err != nil {
		return err
	}
	runA := domain.CaptureRun{ID: "run-a", CarrierID: "carrier-a", SegmentID: "segment-a", Attempt: 1, EquipmentChain: []string{"Studer A810", "Prism ADA-8XR"}, OutputFile: "SC-TAPE-001-segment-a.wav", ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Measurements: domain.SignalMeasurements{PeakDBFS: -3, SilenceMillis: 250, DropoutMillis: 0, TimebasePPM: 2, MeasuredDuration: 20000}}
	runB := domain.CaptureRun{ID: "run-b", CarrierID: "carrier-a", SegmentID: "segment-b", Attempt: 1, EquipmentChain: []string{"Studer A810", "Prism ADA-8XR"}, OutputFile: "SC-TAPE-001-segment-b.wav", ChecksumSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Measurements: domain.SignalMeasurements{PeakDBFS: -3, SilenceMillis: 250, DropoutMillis: 0, TimebasePPM: 2, MeasuredDuration: 20000}}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/capture-runs", map[string]any{"expectedVersion": batch.Version, "actor": "operator-a", "runs": []domain.CaptureRun{runA, runB}}, &batch); err != nil {
		return err
	}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/quality-runs", map[string]any{"expectedVersion": batch.Version, "actor": "operator-a"}, &batch); err != nil {
		return err
	}
	if len(batch.Findings) != 0 {
		return fmt.Errorf("合格测量意外产生 %d 个缺陷", len(batch.Findings))
	}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/review-submissions", map[string]any{"expectedVersion": batch.Version, "actor": "operator-a"}, &batch); err != nil {
		return err
	}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/review-decisions", map[string]any{"expectedVersion": batch.Version, "actor": "reviewer-b", "decision": "approved", "reasonCodes": []string{}, "comment": "自检复核通过"}, &batch); err != nil {
		return err
	}
	if err := c.post(ctx, "/api/v1/batches/selfcheck-batch/certificate", map[string]any{"expectedVersion": batch.Version, "actor": "reviewer-b"}, &batch); err != nil {
		return err
	}
	var verification struct {
		Valid   bool   `json:"valid"`
		Message string `json:"message"`
	}
	if err := c.get(ctx, "/api/v1/batches/selfcheck-batch/certificate/verification", &verification); err != nil {
		return err
	}
	if !verification.Valid {
		return fmt.Errorf("凭据核验未通过: %s", verification.Message)
	}
	var view map[string]any
	if err := c.get(ctx, "/api/v1/batches/selfcheck-batch", &view); err != nil {
		return err
	}
	return nil
}
