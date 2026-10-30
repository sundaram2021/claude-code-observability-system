package cli_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/claude-code/observability/internal/cli"
)

func TestIsRunningAndStartBackground(t *testing.T) {
	// 1. Inactive port
	if cli.IsRunning(59999) {
		t.Errorf("expected inactive port to return false")
	}

	// 2. Active listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	parts := strings.Split(ln.Addr().String(), ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	if !cli.IsRunning(port) {
		t.Errorf("expected active port %d to return true", port)
	}

	// 3. StartBackground when already running
	err = cli.StartBackground(port)
	if err != nil {
		t.Errorf("unexpected error starting background on running port: %v", err)
	}
}

func TestPrintStatus(t *testing.T) {
	// Inactive port
	err := cli.PrintStatus(59998)
	if err != nil {
		t.Errorf("unexpected error for inactive server status: %v", err)
	}

	// Active metrics server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/metrics" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"success": true, "data": {"totalTraces": 5, "totalCostUsd": 0.042}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	parts := strings.Split(server.Listener.Addr().String(), ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	err = cli.PrintStatus(port)
	if err != nil {
		t.Errorf("unexpected error for active server status: %v", err)
	}
}
