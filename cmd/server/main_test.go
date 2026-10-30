package main

import (
	"context"
	"net"
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

