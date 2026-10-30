package handler_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claude-code/observability/internal/agent"
	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/service"
)

type failingRepo struct{}

func (f *failingRepo) SaveTrace(_ *domain.Trace) error                     { return errors.New("err") }
func (f *failingRepo) GetTrace(_ string) (*domain.Trace, error)           { return nil, errors.New("err") }
func (f *failingRepo) ListTraces(_ domain.TraceFilter) ([]*domain.Trace, int, error) {
	return nil, 0, errors.New("err")
}
func (f *failingRepo) SaveSpan(_ *domain.Span) error                      { return errors.New("err") }
func (f *failingRepo) GetSpan(_ string) (*domain.Span, error)             { return nil, errors.New("err") }
func (f *failingRepo) Count() int                                         { return 0 }
func (f *failingRepo) Clear() error                                       { return errors.New("err") }
func (f *failingRepo) SaveSession(_ *domain.Session) error                { return errors.New("err") }
func (f *failingRepo) GetSession(_ string) (*domain.Session, error)       { return nil, errors.New("err") }
func (f *failingRepo) ListSessions(_, _ int) ([]domain.SessionSummary, int, error) {
	return nil, 0, errors.New("err")
}

func TestHandler_ErrorBranches(t *testing.T) {
	errRepo := &failingRepo{}
	traceSvc := service.NewTraceService(errRepo, nil)
	sessSvc := service.NewSessionService(errRepo, errRepo)
	metricsSvc := service.NewMetricsService(errRepo, errRepo)
	runner := agent.NewRunner(traceSvc, "", "")

	traceH := handler.NewTraceHandler(traceSvc, errRepo)
	sessH := handler.NewSessionHandler(sessSvc)
	metricH := handler.NewMetricsHandler(metricsSvc)
	ingestH := handler.NewIngestHandler(traceSvc)
	agentH := handler.NewAgentHandler(runner)

	// 1. Missing path value in GetTrace
	req := httptest.NewRequest(http.MethodGet, "/api/v1/traces/", nil)
	w := httptest.NewRecorder()
	traceH.GetTrace(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty trace ID, got %d", w.Code)
	}

	// 2. Missing path value in GetSession
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/", nil)
	w = httptest.NewRecorder()
	sessH.GetSession(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty session ID, got %d", w.Code)
	}

	// 3. ListTraces error branch
	req = httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
	w = httptest.NewRecorder()
	traceH.ListTraces(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for failing list traces")
	}

	// 4. ClearTraces error branch
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/traces", nil)
	w = httptest.NewRecorder()
	traceH.ClearTraces(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for failing clear traces")
	}

	// 5. ListSessions error branch
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	w = httptest.NewRecorder()
	sessH.ListSessions(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for failing list sessions")
	}

	// 6. GetMetrics error branch
	req = httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	w = httptest.NewRecorder()
	metricH.GetMetrics(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for failing metrics")
	}

	// 7. GetTelemetryLegacy error branch
	req = httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
	w = httptest.NewRecorder()
	ingestH.GetTelemetryLegacy(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for failing telemetry legacy")
	}

	// 8. IngestOTLP ingestion error
	otlpJSON := `{"resourceSpans":[{"scopeSpans":[{"spans":[{"traceId":"t_err","startTimeUnixNano":"100","endTimeUnixNano":"200"}]}]}]}`
	req = httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewBufferString(otlpJSON))
	w = httptest.NewRecorder()
	ingestH.IngestOTLP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for failing OTLP ingest")
	}

	// 9. Agent run error branch (runner fails to ingest trace)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agent/run", bytes.NewBufferString(`{"prompt":"test task"}`))
	w = httptest.NewRecorder()
	agentH.RunAgent(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when runner fails to ingest trace")
	}

	// 10. LegacyChat error branch
	req = httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":"test chat"}`))
	w = httptest.NewRecorder()
	agentH.LegacyChat(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when legacy chat runner fails")
	}
}

