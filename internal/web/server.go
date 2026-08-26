package web

import (
	"embed"
	"net/http"
	"strings"
	"tape-preservation-gate/internal/workflow"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	workflow *workflow.Service
	mux      *http.ServeMux
}

func New(service *workflow.Service) *Server {
	s := &Server{workflow: service, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(s.mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.HealthHandler)
	s.mux.HandleFunc("GET /app", s.AppHandler)
	s.mux.HandleFunc("GET /", s.RootHandler)
	s.mux.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))
	s.mux.HandleFunc("GET /api/v1/batches", s.ListBatchesHandler)
	s.mux.HandleFunc("POST /api/v1/batches", s.CreateBatchHandler)
	s.mux.HandleFunc("GET /api/v1/batches/{batchID}", s.GetBatchHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/carriers", s.AddCarrierHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/carrier-inspections", s.InspectCarrierHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/plan-freeze", s.FreezePlanHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/capture-runs", s.AddCaptureRunHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/quality-runs", s.RunQualityHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/finding-resolutions", s.ResolveFindingHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/review-submissions", s.SubmitReviewHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/review-decisions", s.DecideReviewHandler)
	s.mux.HandleFunc("POST /api/v1/batches/{batchID}/certificate", s.IssueCertificateHandler)
	s.mux.HandleFunc("GET /api/v1/batches/{batchID}/certificate/verification", s.VerifyCertificateHandler)
}

func (s *Server) RootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (s *Server) AppHandler(w http.ResponseWriter, r *http.Request) {
	b, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "页面资源不可用", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}

func batchID(r *http.Request) string { return strings.TrimSpace(r.PathValue("batchID")) }
