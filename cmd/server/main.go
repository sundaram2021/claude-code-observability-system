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
	"github.com/claude-code/observability/internal/cli"
	"github.com/claude-code/observability/internal/config"
	"github.com/claude-code/observability/internal/handler"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
	"github.com/claude-code/observability/web"
)

// BuildServer constructs and wires the HTTP server instance.
func BuildServer(cfg *config.Config) *http.Server {
	return BuildServerWithShutdown(cfg, nil)
}

// BuildServerWithShutdown constructs the server with an optional shutdown hook.
func BuildServerWithShutdown(cfg *config.Config, onShutdown func()) *http.Server {
	store := repository.NewMemoryStore(cfg.MaxTraces, cfg.MaxSessions)
	scoreStore := repository.NewScoreStore(cfg.MaxTraces)

	eventBus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, eventBus)
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)
	scoreSvc := service.NewScoreService(scoreStore, store)

	runner := agent.NewRunner(traceSvc, cfg.AnthropicAPIKey, cfg.DefaultModel)

	router := handler.NewRouter(handler.RouterConfig{
		TraceHandler:   handler.NewTraceHandler(traceSvc, store),
		SessionHandler: handler.NewSessionHandler(sessSvc),
		MetricsHandler: handler.NewMetricsHandler(metricSvc),
		ScoreHandler:   handler.NewScoreHandler(scoreSvc),
		ExportHandler:  handler.NewExportHandler(traceSvc),
		IngestHandler:  handler.NewIngestHandler(traceSvc),
		AgentHandler:   handler.NewAgentHandler(runner),
		SSEHandler:     handler.NewSSEHandler(eventBus),
		StaticFS:       web.GetFileSystem(),
		OnShutdown:     onShutdown,
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

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "start":
			if err := cli.StartBackground(cfg.Port); err != nil {
				log.Fatalf("Failed to start daemon: %v", err)
			}
			_ = cli.OpenBrowser(fmt.Sprintf("http://localhost:%d", cfg.Port))
			return
		case "stop":
			_ = cli.Stop(cfg.Port)
			return
		case "status":
			_ = cli.PrintStatus(cfg.Port)
			return
		case "open":
			_ = cli.OpenBrowser(fmt.Sprintf("http://localhost:%d", cfg.Port))
			return
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	srv := BuildServerWithShutdown(cfg, func() {
		quit <- os.Interrupt
	})
	go func() {
		log.Printf("==========================================================")
		log.Printf(" Claude Code Observability (Langfuse for Claude Code)")
		log.Printf(" Dashboard UI: http://localhost:%d", cfg.Port)
		log.Printf("==========================================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	<-quit

	log.Println("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("Server exited cleanly.")
}
