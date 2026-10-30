package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// IsRunning performs a fast TCP check to verify if the server is active.
func IsRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// StartBackground launches the observability server detached in background.
func StartBackground(port int) error {
	if IsRunning(port) {
		fmt.Printf("✓ Claude Code Observability is already running on http://localhost:%d\n", port)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		exe = "server.exe"
	}

	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000 | 0x00000200, // CREATE_NO_WINDOW | CREATE_NEW_PROCESS_GROUP
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start background daemon: %w", err)
	}

	// Wait up to 1 second for port to be ready
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		if IsRunning(port) {
			fmt.Printf("✓ Started Claude Code Observability daemon on http://localhost:%d (PID %d)\n", port, cmd.Process.Pid)
			return nil
		}
	}
	return fmt.Errorf("server launched but port %d did not respond", port)
}

// OpenBrowser launches the user's default browser to the given URL.
func OpenBrowser(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", targetURL)
	case "darwin":
		cmd = exec.Command("open", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	return cmd.Start()
}

// PrintStatus queries and displays active telemetry summary from the running daemon.
func PrintStatus(port int) error {
	if !IsRunning(port) {
		fmt.Printf("✗ Claude Code Observability is stopped (port %d inactive)\n", port)
		return nil
	}

	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/metrics", port))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var payload struct {
		Success bool                 `json:"success"`
		Data    domain.SystemMetrics `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return err
	}

	m := payload.Data
	fmt.Printf("========================================================\n")
	fmt.Printf(" Claude Code Observability Status (http://localhost:%d)\n", port)
	fmt.Printf("========================================================\n")
	fmt.Printf(" Status:       RUNNING\n")
	fmt.Printf(" Total Traces: %d\n", m.TotalTraces)
	fmt.Printf(" Total Spans:  %d\n", m.TotalSpans)
	fmt.Printf(" Total Cost:   $%.4f (Saved $%.4f via prompt cache)\n", m.TotalCostUSD, m.TotalCacheSavingsUSD)
	fmt.Printf(" Cache Hit %%:  %.1f%%\n", m.CacheHitRate)
	fmt.Printf(" Avg Latency:  %.0fms (P95: %.0fms)\n", m.AvgLatencyMs, m.P95LatencyMs)
	fmt.Printf(" Error Rate:   %.1f%%\n", m.ErrorRate)
	fmt.Printf("========================================================\n")
	return nil
}
