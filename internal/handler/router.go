package handler

import (
	"log"
	"net/http"
	"time"
)

// RouterConfig contains dependencies for initializing the HTTP router.
type RouterConfig struct {
	TraceHandler   *TraceHandler
	SessionHandler *SessionHandler
	MetricsHandler *MetricsHandler
	ScoreHandler   *ScoreHandler
	ExportHandler  *ExportHandler
	IngestHandler  *IngestHandler
	AgentHandler   *AgentHandler
	SSEHandler     *SSEHandler
	StaticFS       http.FileSystem
	OnShutdown     func()
}

// NewRouter constructs and configures the main HTTP multiplexer with middleware.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Traces API
	if cfg.TraceHandler != nil {
		mux.HandleFunc("GET /api/v1/traces", cfg.TraceHandler.ListTraces)
		mux.HandleFunc("GET /api/v1/traces/{id}", cfg.TraceHandler.GetTrace)
		mux.HandleFunc("POST /api/v1/traces", cfg.TraceHandler.CreateTrace)
		mux.HandleFunc("DELETE /api/v1/traces", cfg.TraceHandler.ClearTraces)
	}

	// Export API
	if cfg.ExportHandler != nil {
		mux.HandleFunc("GET /api/v1/traces/export", cfg.ExportHandler.ExportTraces)
	}

	// Scores & Evaluations API
	if cfg.ScoreHandler != nil {
		mux.HandleFunc("POST /api/v1/scores", cfg.ScoreHandler.CreateScore)
		mux.HandleFunc("GET /api/v1/scores", cfg.ScoreHandler.ListScores)
		mux.HandleFunc("GET /api/v1/traces/{id}/scores", cfg.ScoreHandler.GetTraceScores)
	}

	// Sessions API
	if cfg.SessionHandler != nil {
		mux.HandleFunc("GET /api/v1/sessions", cfg.SessionHandler.ListSessions)
		mux.HandleFunc("GET /api/v1/sessions/{id}", cfg.SessionHandler.GetSession)
	}

	// Metrics API
	if cfg.MetricsHandler != nil {
		mux.HandleFunc("GET /api/v1/metrics", cfg.MetricsHandler.GetMetrics)
	}

	// Ingestion Endpoints (OTLP, Claude Hooks, Langfuse, Legacy)
	if cfg.IngestHandler != nil {
		mux.HandleFunc("POST /v1/traces", cfg.IngestHandler.IngestOTLP)
		mux.HandleFunc("POST /v1/logs", cfg.IngestHandler.IngestOTLPLogs)
		mux.HandleFunc("POST /v1/metrics", cfg.IngestHandler.IngestOTLPMetrics)
		mux.HandleFunc("POST /api/v1/events", cfg.IngestHandler.IngestClaudeEvent)
		mux.HandleFunc("POST /api/public/ingestion", cfg.IngestHandler.IngestLangfuse)
		mux.HandleFunc("GET /api/telemetry", cfg.IngestHandler.GetTelemetryLegacy)
	}

	// Agent & Chat Endpoints
	if cfg.AgentHandler != nil {
		mux.HandleFunc("POST /api/v1/agent/run", cfg.AgentHandler.RunAgent)
		mux.HandleFunc("POST /api/chat", cfg.AgentHandler.LegacyChat)
	}

	// Realtime SSE
	if cfg.SSEHandler != nil {
		mux.Handle("GET /api/v1/live", cfg.SSEHandler)
	}

	// Graceful Shutdown API
	if cfg.OnShutdown != nil {
		mux.HandleFunc("POST /api/v1/shutdown", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"message":"server shutting down"}`))
			go func() {
				time.Sleep(50 * time.Millisecond)
				cfg.OnShutdown()
			}()
		})
	}

	// Static UI Dashboard
	if cfg.StaticFS != nil {
		mux.Handle("/", http.FileServer(cfg.StaticFS))
	}

	return withMiddleware(mux)
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, rec)
				WriteError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)

		if r.URL.Path != "/api/v1/live" {
			log.Printf("[%s] %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
		}
	})
}
