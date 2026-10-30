package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claude-code/observability/internal/agent"
	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
	"github.com/claude-code/observability/web"
)

func TestFullE2EWorkflow(t *testing.T) {
	store := repository.NewMemoryStore(100, 100)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)
	runner := agent.NewRunner(traceSvc, "", "claude-3-7-sonnet")

	router := handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(traceSvc, store),
		SessionHandler: handler.NewSessionHandler(sessSvc),
		MetricsHandler: handler.NewMetricsHandler(metricSvc),
		IngestHandler:  handler.NewIngestHandler(traceSvc),
		AgentHandler:   handler.NewAgentHandler(runner),
		SSEHandler:     handler.NewSSEHandler(bus),
		StaticFS:       web.GetFileSystem(),
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// 1. Test Static UI is served
	res, err := http.Get(server.URL + "/")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("failed to serve static UI: %v", err)
	}

	// 2. Test Agent Task Simulation
	simPayload := `{"prompt": "Run tests and refactor trace service", "model": "claude-3-7-sonnet", "simulate": true}`
	res, err = http.Post(server.URL+"/api/v1/agent/run", "application/json", bytes.NewBufferString(simPayload))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("agent run simulation failed: %v", err)
	}

	// 3. Test Claude Code Hook Ingestion
	hookPayload := `{
		"sessionId": "sess_e2e_cli",
		"project": "claude-code-project",
		"turnId": "turn_e2e_01",
		"prompt": "Fix bug in memory store",
		"toolName": "file_edit",
		"toolInput": "{\"file\":\"memory_store.go\"}",
		"toolOutput": "Success",
		"durationMs": 120.0,
		"tokens": {"inputTokens": 600, "outputTokens": 150, "cacheReadTokens": 3000}
	}`
	res, err = http.Post(server.URL+"/api/v1/events", "application/json", bytes.NewBufferString(hookPayload))
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("hook event ingestion failed: %v", err)
	}

	// 4. Test List Traces
	res, err = http.Get(server.URL + "/api/v1/traces")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("list traces failed: %v", err)
	}
	var tracesResp struct {
		Success bool            `json:"success"`
		Data    []*domain.Trace `json:"data"`
		Total   int             `json:"total"`
	}
	_ = json.NewDecoder(res.Body).Decode(&tracesResp)
	if tracesResp.Total < 2 {
		t.Errorf("expected at least 2 traces, got %d", tracesResp.Total)
	}

	// 5. Test Metrics and Cache Savings
	res, err = http.Get(server.URL + "/api/v1/metrics")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("get metrics failed: %v", err)
	}
	var metricsResp struct {
		Success bool                 `json:"success"`
		Data    domain.SystemMetrics `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&metricsResp)
	m := metricsResp.Data
	if m.TotalCostUSD <= 0 {
		t.Errorf("expected total cost > 0, got %f", m.TotalCostUSD)
	}
	if m.Tokens.CacheReadInputTokens <= 0 {
		t.Errorf("expected cache read tokens > 0, got %d", m.Tokens.CacheReadInputTokens)
	}
	if len(m.Tools) == 0 {
		t.Errorf("expected recorded tools in metrics, got 0")
	}

	// 6. Test Sessions
	res, err = http.Get(server.URL + "/api/v1/sessions")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("get sessions failed: %v", err)
	}
}

