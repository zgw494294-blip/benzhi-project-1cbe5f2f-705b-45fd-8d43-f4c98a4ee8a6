package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
	"tape-preservation-gate/internal/workflow"
	"testing"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	r, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(workflow.New(r, quality.NewEngine())).Handler()
}

func TestAppAndSecurityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	rec := httptest.NewRecorder()
	testHandler(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<body>") {
		t.Fatalf("工作台页面不可用: %d", rec.Code)
	}
	if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("缺少安全响应头")
	}
}

func TestCreateAndQueryBatchAPI(t *testing.T) {
	h := testHandler(t)
	body := `{"id":"api-batch","title":"接口测试","operator":"op","reviewer":"rv","actor":"op","targetProfile":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建失败: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/batches/api-batch", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "allowedActions") {
		t.Fatalf("详情查询失败: %d %s", rec.Code, rec.Body.String())
	}
}
