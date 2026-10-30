package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration for the Claude Code observability server.
type Config struct {
	Host            string
	Port            int
	MaxTraces       int
	MaxSessions     int
	AnthropicAPIKey string
	DefaultModel    string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

// Load loads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Host:            getEnv("HOST", "0.0.0.0"),
		Port:            getEnvAsInt("PORT", 3000),
		MaxTraces:       getEnvAsInt("MAX_TRACES", 5000),
		MaxSessions:     getEnvAsInt("MAX_SESSIONS", 1000),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		DefaultModel:    getEnv("DEFAULT_MODEL", "claude-3-7-sonnet-20250219"),
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
	}
}

// getEnv retrieves an environment variable or returns the default value.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// getEnvAsInt retrieves an environment variable parsed as an integer or returns the default.
func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

