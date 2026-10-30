package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestScoreHandler_AndExport(t *testing.T) {
	traceStore := repository.NewMemoryStore(10, 10)
	scoreStore := repository.NewScoreStore(10)
	traceSvc := service.NewTraceService(traceStore, nil)
	scoreSvc := service.NewScoreService(scoreStore, traceStore)

	scoreH := handler.NewScoreHandler(scoreSvc)
	exportH := handler.NewExportHandler(traceSvc)

	router := handler.NewRouter(handler.RouterConfig{
		TraceHandler:  handler.NewTraceHandler(traceSvc, traceStore),
		ScoreHandler:  scoreH,
		ExportHandler: exportH,
	})

	_ = traceSvc.IngestTrace(&domain.Trace{
		ID:   "trc_target_1",
		Name: "Exportable trace",
	})

	// 1. POST /api/v1/scores valid
	scoreJSON := `{"traceId":"trc_target_1","name":"accuracy","value":0.9,"comment":"accurate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scores", bytes.NewBufferString(scoreJSON))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for score, got %d", w.Code)
	}

	// 2. POST /api/v1/scores malformed JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/scores", bytes.NewBufferString(`{bad-json`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad score json, got %d", w.Code)
	}

	// 3. POST /api/v1/scores missing required fields
	req = httptest.NewRequest(http.MethodPost, "/api/v1/scores", bytes.NewBufferString(`{"value":0.5}`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing traceId, got %d", w.Code)
	}

	// 4. GET /api/v1/scores all
	req = httptest.NewRequest(http.MethodGet, "/api/v1/scores", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for list scores, got %d", w.Code)
	}

	// 5. GET /api/v1/scores?traceId=trc_target_1
	req = httptest.NewRequest(http.MethodGet, "/api/v1/scores?traceId=trc_target_1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for filtered scores, got %d", w.Code)
	}

	// 6. GET /api/v1/traces/{id}/scores
	req = httptest.NewRequest(http.MethodGet, "/api/v1/traces/trc_target_1/scores", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for trace scores, got %d", w.Code)
	}

	// 7. GET /api/v1/traces/{id}/scores with missing id
	req = httptest.NewRequest(http.MethodGet, "/api/v1/traces//scores", nil)
	w = httptest.NewRecorder()
	scoreH.GetTraceScores(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty trace ID in GetTraceScores")
	}

	// 8. GET /api/v1/traces/export
	req = httptest.NewRequest(http.MethodGet, "/api/v1/traces/export", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for trace export, got %d", w.Code)
	}
}
