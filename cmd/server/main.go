package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
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
	srv, _ := BuildServerWithShutdown(cfg, nil)
	return srv
}

// BuildServerWithShutdown constructs the server with an optional shutdown hook and returns its EventBus.
func BuildServerWithShutdown(cfg *config.Config, onShutdown func()) (*http.Server, *service.EventBus) {
	debug.SetGCPercent(20)
	debug.SetMemoryLimit(12 * 1024 * 1024)

	store := repository.NewMemoryStore(cfg.MaxTraces, cfg.MaxSessions)
	if loaded, _ := repository.LoadFromFile(store, ""); loaded > 0 {
		log.Printf("Loaded %d traces from disk", loaded)
	}
	scoreStore := repository.NewScoreStore(cfg.MaxTraces)

	eventBus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, eventBus)
	traceSvc.SetPersistencePath(repository.DefaultTraceFilePath())
	sessSvc := service.NewSessionService(store, store)
	metricSvc := service.NewMetricsService(store, store)
	scoreSvc := service.NewScoreService(scoreStore, store)

	runner := agent.NewRunner(traceSvc, cfg.AnthropicAPIKey, cfg.DefaultModel)

	wrappedShutdown := func() {
		_ = repository.SaveToFile(store, "")
		if onShutdown != nil {
			onShutdown()
		}
	}

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
		OnShutdown:     wrappedShutdown,
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      0, // Streaming SSE / live events require zero write deadline
	}
	return srv, eventBus
}

func main() {
	cfg := config.Load()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "start":
			if err := cli.StartBackground(cfg.Port); err != nil {
				log.Fatalf("Failed to start daemon: %v", err)
			}
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

	srv, _ := BuildServerWithShutdown(cfg, func() {
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
