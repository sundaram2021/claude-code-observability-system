package main

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/claude-code/observability/internal/config"
)

func TestBuildServer(t *testing.T) {
	cfg := config.Load()
	cfg.Port = 0 // random port
	srv := BuildServer(cfg)

	if srv == nil {
		t.Fatalf("expected non-nil server")
	}
	if srv.Handler == nil {
		t.Fatalf("expected non-nil handler")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("error during server shutdown: %v", err)
	}
}

func TestBuildServerWithShutdown(t *testing.T) {
	cfg := config.Load()
	cfg.Port = 0
	called := false
	srv, bus := BuildServerWithShutdown(cfg, func() {
		called = true
	})
	if srv == nil || bus == nil {
		t.Fatalf("expected non-nil server and bus")
	}
	_ = called
}

func TestStartParentWatchdog_Termination(t *testing.T) {
	quit := make(chan os.Signal, 1)

	go func() {
		runParentWatchdog(99999999, nil, 10*time.Millisecond, 2, quit)
	}()

	select {
	case sig := <-quit:
		if sig != os.Interrupt {
			t.Errorf("expected os.Interrupt, got %v", sig)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("watchdog failed to detect dead parent within 500ms")
	}
}


