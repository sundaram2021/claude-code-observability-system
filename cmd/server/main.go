package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/claude-code/observability/internal/agent"
	"github.com/claude-code/observability/internal/config"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
	"github.com/claude-code/observability/web"
)

// BuildServer constructs and wires the HTTP server instance.
func BuildServer(cfg *config.Config) *http.Server {
	store := repository.NewMemoryStore(cfg.MaxTraces, cfg.MaxSessions)

	eventBus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, eventBus)
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)

	runner := agent.NewRunner(traceSvc, cfg.AnthropicAPIKey, cfg.DefaultModel)

	router := handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(traceSvc, store),
		SessionHandler: handler.NewSessionHandler(sessSvc),
		MetricsHandler: handler.NewMetricsHandler(metricSvc),
		IngestHandler:  handler.NewIngestHandler(traceSvc),
		AgentHandler:   handler.NewAgentHandler(runner),
		SSEHandler:     handler.NewSSEHandler(eventBus),
		StaticFS:       web.GetFileSystem(),
	})

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}

func main() {
	cfg := config.Load()
	srv := BuildServer(cfg)

	go func() {
		log.Printf("==========================================================")
		log.Printf(" Claude Code Observability (Langfuse for Claude Code)")
		log.Printf(" Dashboard UI: http://localhost:%d", cfg.Port)
		log.Printf("==========================================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly.")
}
