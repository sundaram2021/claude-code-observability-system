package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func setupTestRouter() http.Handler {
	store := repository.NewMemoryStore(100, 100)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)

	return handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(traceSvc, store),
		SessionHandler: handler.NewSessionHandler(sessSvc),
		MetricsHandler: handler.NewMetricsHandler(metricSvc),
		IngestHandler:  handler.NewIngestHandler(traceSvc),
		AgentHandler:   handler.NewAgentHandler(nil),
		SSEHandler:     handler.NewSSEHandler(bus),
	})
}

func TestIngestHandler_OTLPEndpoint(t *testing.T) {
	router := setupTestRouter()

	otlpJSON := `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "claude-code"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "trace_http_01",
					"spanId": "span_http_01",
					"name": "agent.task",
					"startTimeUnixNano": "1700000000000000000",
					"endTimeUnixNano": "1700000000200000000",
					"attributes": [
						{"key": "gen_ai.request.model", "value": {"stringValue": "claude-3-7-sonnet"}},
						{"key": "gen_ai.usage.input_tokens", "value": {"intValue": 500}},
						{"key": "gen_ai.usage.output_tokens", "value": {"intValue": 100}}
					]
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewBufferString(otlpJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// Now query /api/v1/traces and verify the ingested trace is available
	queryReq := httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
	queryW := httptest.NewRecorder()
	router.ServeHTTP(queryW, queryReq)

	if queryW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on listing traces, got %d", queryW.Code)
	}

	if !bytes.Contains(queryW.Body.Bytes(), []byte("trace_http_01")) {
		t.Errorf("expected trace_http_01 in traces list output: %s", queryW.Body.String())
	}
}

func TestIngestHandler_ClaudeEventEndpoint(t *testing.T) {
	router := setupTestRouter()

	claudeEventJSON := `{
		"sessionId": "sess_unit_test",
		"project": "claude-repo",
		"turnId": "turn_unit_99",
		"prompt": "Run unit tests",
		"toolName": "bash",
		"toolInput": "{\"command\": \"go test\"}",
		"toolOutput": "PASS",
		"durationMs": 150.0,
		"tokens": {
			"inputTokens": 300,
			"outputTokens": 50,
			"cacheReadTokens": 1200
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(claudeEventJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// Verify metrics endpoint reflects the ingested tokens and tools
	metricReq := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	metricW := httptest.NewRecorder()
	router.ServeHTTP(metricW, metricReq)

	if metricW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from metrics, got %d", metricW.Code)
	}

	if !bytes.Contains(metricW.Body.Bytes(), []byte("bash")) {
		t.Errorf("expected 'bash' tool in metrics: %s", metricW.Body.String())
	}
}

func TestIngestHandler_OTLPLogsAndMetrics(t *testing.T) {
	router := setupTestRouter()

	reqLogs := httptest.NewRequest(http.MethodPost, "/v1/logs", bytes.NewBufferString(`{}`))
	reqLogs.Header.Set("Content-Type", "application/json")
	wLogs := httptest.NewRecorder()
	router.ServeHTTP(wLogs, reqLogs)

	if wLogs.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /v1/logs, got %d", wLogs.Code)
	}

	reqMetrics := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewBufferString(`{}`))
	reqMetrics.Header.Set("Content-Type", "application/json")
	wMetrics := httptest.NewRecorder()
	router.ServeHTTP(wMetrics, reqMetrics)

	if wMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /v1/metrics, got %d", wMetrics.Code)
	}
}


