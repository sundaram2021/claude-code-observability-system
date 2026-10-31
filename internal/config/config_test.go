package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/claude-code/observability/internal/config"
)

func TestConfig_DefaultValues(t *testing.T) {
	// Ensure relevant env vars are clean
	os.Unsetenv("HOST")
	os.Unsetenv("PORT")
	os.Unsetenv("MAX_TRACES")
	os.Unsetenv("MAX_SESSIONS")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("DEFAULT_MODEL")

	cfg := config.Load()

	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.Port != 3456 {
		t.Errorf("expected port 3456, got %d", cfg.Port)
	}
	if cfg.MaxTraces != 5000 {
		t.Errorf("expected max traces 5000, got %d", cfg.MaxTraces)
	}
	if cfg.MaxSessions != 1000 {
		t.Errorf("expected max sessions 1000, got %d", cfg.MaxSessions)
	}
	if cfg.DefaultModel != "claude-3-7-sonnet-20250219" {
		t.Errorf("expected default model, got %s", cfg.DefaultModel)
	}
	if cfg.ReadTimeout != 15*time.Second || cfg.WriteTimeout != 15*time.Second {
		t.Errorf("expected 15s timeouts")
	}
}

func TestConfig_CustomEnvValues(t *testing.T) {
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("PORT", "8080")
	os.Setenv("MAX_TRACES", "250")
	os.Setenv("MAX_SESSIONS", "50")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	os.Setenv("DEFAULT_MODEL", "claude-3-5-haiku-20241022")

	defer func() {
		os.Unsetenv("HOST")
		os.Unsetenv("PORT")
		os.Unsetenv("MAX_TRACES")
		os.Unsetenv("MAX_SESSIONS")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("DEFAULT_MODEL")
	}()

	cfg := config.Load()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.MaxTraces != 250 {
		t.Errorf("expected max traces 250, got %d", cfg.MaxTraces)
	}
	if cfg.MaxSessions != 50 {
		t.Errorf("expected max sessions 50, got %d", cfg.MaxSessions)
	}
	if cfg.AnthropicAPIKey != "sk-ant-test-key" {
		t.Errorf("expected api key sk-ant-test-key, got %s", cfg.AnthropicAPIKey)
	}
	if cfg.DefaultModel != "claude-3-5-haiku-20241022" {
		t.Errorf("expected custom model, got %s", cfg.DefaultModel)
	}
}

func TestConfig_InvalidIntEnvFallback(t *testing.T) {
	os.Setenv("PORT", "not-a-number")
	defer os.Unsetenv("PORT")

	cfg := config.Load()
	if cfg.Port != 3456 {
		t.Errorf("expected port fallback 3456, got %d", cfg.Port)
	}
}

