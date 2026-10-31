package ingestion

import (
	"strconv"

	"github.com/claude-code/observability/internal/domain"
)

// buildToolCall extracts and formats tool execution details from attributes.
func buildToolCall(attrs map[string]string) *domain.ToolCall {
	toolName := attrs["tool_name"]
	if toolName == "" {
		toolName = attrs["tool"]
	}
	if toolName == "" {
		return nil
	}

	exitCode, _ := strconv.Atoi(attrs["tool_exit_code"])
	isError := attrs["is_error"] == "true" || exitCode != 0

	return &domain.ToolCall{
		Name:     toolName,
		Input:    attrs["tool_input"],
		Output:   attrs["tool_result"],
		ExitCode: exitCode,
		IsError:  isError,
	}
}

// computeGenVelocity calculates streaming tokens per second.
func computeGenVelocity(durMs int, ttftMs float64, outputTokens int) float64 {
	if durMs <= 0 || outputTokens <= 0 {
		return 0
	}
	activeSec := (float64(durMs) - ttftMs) / 1000.0
	if activeSec <= 0.05 {
		activeSec = float64(durMs) / 1000.0
	}
	if activeSec > 0 {
		return float64(outputTokens) / activeSec
	}
	return 0
}

// enrichContextTokens computes total context tokens and saturation percentage.
func enrichContextTokens(u *domain.TokenUsage, attrs map[string]string) {
	u.CacheCreation5mTokens, _ = strconv.Atoi(attrs["cache_creation_5m_tokens"])
	u.CacheCreation1hTokens, _ = strconv.Atoi(attrs["cache_creation_1h_tokens"])
	u.ContextTokens = u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
	u.ContextWindowLimit = 200000
	if u.ContextTokens > 0 {
		u.ContextSaturationPct = (float64(u.ContextTokens) / 200000.0) * 100.0
	}
}

