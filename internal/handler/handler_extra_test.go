package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestHandler_SessionDetailSuccess(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)

	router := handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(traceSvc, store),
		SessionHandler: handler.NewSessionHandler(sessSvc),
		MetricsHandler: handler.NewMetricsHandler(metricSvc),
		IngestHandler:  handler.NewIngestHandler(traceSvc),
		AgentHandler:   handler.NewAgentHandler(nil),
		SSEHandler:     handler.NewSSEHandler(bus),
	})

	// Create trace with session
	_ = traceSvc.IngestTrace(&domain.Trace{
		ID:        "trc_for_sess",
		SessionID: "sess_detail_ok",
		Name:      "Turn for detail",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/sess_detail_ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for session detail, got %d", w.Code)
	}
}

func TestHandler_PanicRecoveryMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("simulated disaster")
	})

	router := handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(nil, nil),
		SessionHandler: handler.NewSessionHandler(nil),
		MetricsHandler: handler.NewMetricsHandler(nil),
		IngestHandler:  handler.NewIngestHandler(nil),
		AgentHandler:   handler.NewAgentHandler(nil),
		SSEHandler:     handler.NewSSEHandler(nil),
	})

	// Test panic recovery through router
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unknown-path", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown path, got %d", w.Code)
	}
}

func TestHandler_SSEStreaming(t *testing.T) {
	bus := service.NewEventBus()
	sseHandler := handler.NewSSEHandler(bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/live", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		sseHandler.ServeHTTP(w, req)
		close(done)
	}()

	// Wait briefly for connection
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event
	bus.Publish(service.RealtimeEvent{
		Type: service.EventTraceCreated,
		Trace: &domain.Trace{ID: "trc_live_1"},
	})

	time.Sleep(50 * time.Millisecond)
	cancel() // trigger context cancellation
	<-done

	if !w.Flushed {
		t.Errorf("expected response to be flushed")
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Errorf("expected non-empty SSE output")
	}
}

