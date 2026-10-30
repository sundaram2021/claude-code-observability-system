package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claude-code/observability/internal/agent"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func setupFullHandlerRouter() http.Handler {
	store := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)
	runner := agent.NewRunner(traceSvc, "", "claude-3-7-sonnet")

	return handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(traceSvc, store),
		SessionHandler: handler.NewSessionHandler(sessSvc),
		MetricsHandler: handler.NewMetricsHandler(metricSvc),
		IngestHandler:  handler.NewIngestHandler(traceSvc),
		AgentHandler:   handler.NewAgentHandler(runner),
		SSEHandler:     handler.NewSSEHandler(bus),
	})
}

func TestHandler_TraceEndpoints(t *testing.T) {
	router := setupFullHandlerRouter()

	// 1. GetTrace not found
	req := httptest.NewRequest(http.MethodGet, "/api/v1/traces/non_existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// 2. CreateTrace malformed
	req = httptest.NewRequest(http.MethodPost, "/api/v1/traces", bytes.NewBufferString("{bad-json"))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", w.Code)
	}

	// 3. CreateTrace valid
	validTrace := `{"id":"trc_manual_01","name":"Manual Trace"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/traces", bytes.NewBufferString(validTrace))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", w.Code)
	}

	// 4. GetTrace found
	req = httptest.NewRequest(http.MethodGet, "/api/v1/traces/trc_manual_01", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	// 5. Clear traces
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/traces", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for delete, got %d", w.Code)
	}
}

func TestHandler_SessionAndAgentEndpoints(t *testing.T) {
	router := setupFullHandlerRouter()

	// 1. Session not found
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/missing_sess", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d", w.Code)
	}

	// 2. Agent run missing prompt
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agent/run", bytes.NewBufferString(`{"prompt":""}`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing prompt, got %d", w.Code)
	}

	// 3. Agent run malformed JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agent/run", bytes.NewBufferString(`bad json`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed json, got %d", w.Code)
	}

	// 4. Legacy Chat valid
	chatBody := `{"message":"hello claude"}`
	req = httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(chatBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for legacy chat, got %d", w.Code)
	}

	// 5. Legacy Chat empty message
	req = httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":""}`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty chat message, got %d", w.Code)
	}

	// 6. CORS OPTIONS Preflight
	req = httptest.NewRequest(http.MethodOptions, "/api/v1/traces", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for OPTIONS, got %d", w.Code)
	}
}

func TestHandler_IngestEdgeCases(t *testing.T) {
	router := setupFullHandlerRouter()

	// 1. OTLP bad json
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewBufferString(`invalid`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad OTLP json")
	}

	// 2. Claude event bad json
	req = httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(`invalid`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad claude event json")
	}

	// 3. Langfuse bad json
	req = httptest.NewRequest(http.MethodPost, "/api/public/ingestion", bytes.NewBufferString(`invalid`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad langfuse json")
	}

	// 4. Legacy telemetry GET
	req = httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for legacy telemetry")
	}
}

