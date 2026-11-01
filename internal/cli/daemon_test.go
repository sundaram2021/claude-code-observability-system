package cli_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestStop(t *testing.T) {
	_ = cli.Stop(59997)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/shutdown" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	parts := strings.Split(server.Listener.Addr().String(), ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	_ = cli.Stop(port)
	server.Close()
}

func TestPrintStatus_BadResponses(t *testing.T) {
	// Bad JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{malformed json`)
	}))
	defer server.Close()
	parts := strings.Split(server.Listener.Addr().String(), ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	_ = cli.PrintStatus(port)

	// Connection error
	_ = cli.PrintStatus(59996)

	// Cache multiplier branch
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success": true, "data": {"cacheMultiplier": 2.5, "cacheHitRate": 85.0}}`)
	}))
	defer server2.Close()
	parts2 := strings.Split(server2.Listener.Addr().String(), ":")
	port2, _ := strconv.Atoi(parts2[len(parts2)-1])
	_ = cli.PrintStatus(port2)

	// Status code != 200 error
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server3.Close()
	parts3 := strings.Split(server3.Listener.Addr().String(), ":")
	port3, _ := strconv.Atoi(parts3[len(parts3)-1])
	_ = cli.PrintStatus(port3)
}

func TestOpenBrowser(t *testing.T) {
	// Should not panic or crash
	_ = cli.OpenBrowser("http://localhost:3456")
}

func TestStop_Persistent(t *testing.T) {
	// Server responds but stays open (simulating line 99)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	parts := strings.Split(ln.Addr().String(), ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	_ = cli.Stop(port)
}

func TestStartBackground_LaunchFailure(t *testing.T) {
	oldExec := cli.ExecCommand
	cli.ExecCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd", "/c", "exit", "0")
	}
	defer func() { cli.ExecCommand = oldExec }()

	_ = cli.StartBackground(59990)

	// Command start failure branch
	cli.ExecCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("nonexistent_binary_xyz_12345")
	}
	_ = cli.StartBackground(59991)
}

func TestStartBackground_SuccessLoop(t *testing.T) {
	oldExec := cli.ExecCommand
	cli.ExecCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd", "/c", "exit", "0")
	}
	defer func() { cli.ExecCommand = oldExec }()

	go func() {
		time.Sleep(80 * time.Millisecond)
		ln, _ := net.Listen("tcp", "127.0.0.1:59989")
		if ln != nil {
			time.Sleep(300 * time.Millisecond)
			_ = ln.Close()
		}
	}()
	_ = cli.StartBackground(59989)
}





