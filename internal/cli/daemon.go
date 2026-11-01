package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// ExecCommand wraps exec.Command for testing.
var ExecCommand = exec.Command

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

	cmd := ExecCommand(exe)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	if ppid := os.Getenv("CLAUDE_CODE_PID"); ppid != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CLAUDE_CODE_PID=%s", ppid))
	}
	cmd.Stdin = nil
	logPath := filepath.Join(os.TempDir(), "claude-observability.log")
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	detachProcess(cmd)

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

// Stop gracefully terminates the daemon running on the given port.
func Stop(port int) error {
	if !IsRunning(port) {
		fmt.Printf("Claude Code Observability is not running on port %d\n", port)
		return nil
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/api/v1/shutdown", port), nil)
	_, _ = client.Do(req)
	time.Sleep(100 * time.Millisecond)
	if !IsRunning(port) {
		fmt.Printf("✓ Claude Code Observability stopped on port %d\n", port)
		return nil
	}
	fmt.Printf("✓ Shutdown signal sent to port %d\n", port)
	return nil
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
	fmt.Printf(" Total Traces: %d (Spans: %d)\n", m.TotalTraces, m.TotalSpans)
	fmt.Printf(" Total Cost:   $%.4f (Saved $%.4f via prompt cache)\n", m.TotalCostUSD, m.TotalCacheSavingsUSD)
	if m.CacheMultiplier > 0 {
		fmt.Printf(" Cache ROI:    %.1fx dollar multiplier (Hit Rate: %.1f%%)\n", m.CacheMultiplier, m.CacheHitRate)
	} else {
		fmt.Printf(" Cache Hit %%:  %.1f%%\n", m.CacheHitRate)
	}
	fmt.Printf(" TTFT & Speed: %.0fms TTFT (%.1f tok/s)\n", m.AvgTTFTMs, m.AvgTokensPerSec)
	fmt.Printf(" Context Sat:  %.1f%% of 200k limit\n", m.AvgContextSaturation)
	fmt.Printf(" Avg Latency:  %.0fms (P95: %.0fms)\n", m.AvgLatencyMs, m.P95LatencyMs)
	fmt.Printf(" Resilience:   %.1f%% self-correction (Error Rate: %.1f%%)\n", m.SelfCorrectionRate, m.ErrorRate)
	fmt.Printf("========================================================\n")
	return nil
}
