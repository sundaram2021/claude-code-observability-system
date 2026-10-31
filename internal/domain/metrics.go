package domain

// ToolUsageStat tracks execution counts, duration, and failures per tool.
type ToolUsageStat struct {
	Name          string  `json:"name"`
	CallCount     int     `json:"callCount"`
	AvgDurationMs float64 `json:"avgDurationMs"`
	ErrorCount    int     `json:"errorCount"`
}

// ModelUsageStat tracks usage and expenditure per LLM model.
type ModelUsageStat struct {
	Model       string  `json:"model"`
	Calls       int     `json:"calls"`
	CostUSD     float64 `json:"costUsd"`
	TotalTokens int     `json:"totalTokens"`
}

// SystemMetrics aggregates overall platform statistics for Claude Code observability.
type SystemMetrics struct {
	TotalTraces          int              `json:"totalTraces"`
	TotalSpans           int              `json:"totalSpans"`
	TotalSessions        int              `json:"totalSessions"`
	AvgLatencyMs         float64          `json:"avgLatencyMs"`
	P95LatencyMs         float64          `json:"p95LatencyMs"`
	AvgTTFTMs            float64          `json:"avgTtftMs"`
	AvgTokensPerSec      float64          `json:"avgTokensPerSec"`
	AvgContextSaturation float64          `json:"avgContextSaturation"`
	SelfCorrectionRate   float64          `json:"selfCorrectionRate"`
	ToolDurationMs       float64          `json:"toolDurationMs"`
	GenerationDurationMs float64          `json:"generationDurationMs"`
	TotalCostUSD         float64          `json:"totalCostUsd"`
	TotalCacheSavingsUSD float64          `json:"totalCacheSavingsUsd"`
	TotalUncachedCostUSD float64          `json:"totalUncachedCostUsd"`
	CacheMultiplier      float64          `json:"cacheMultiplier"`
	Tokens               TokenUsage       `json:"tokens"`
	CacheHitRate         float64          `json:"cacheHitRate"`
	ErrorCount           int              `json:"errorCount"`
	ErrorRate            float64          `json:"errorRate"`
	Tools                []ToolUsageStat  `json:"tools"`
	Models               []ModelUsageStat `json:"models"`
}

// AddSpan incorporates a single span into running metrics.
func (sm *SystemMetrics) AddSpan(s *Span) {
	sm.TotalSpans++
	sm.Tokens.InputTokens += s.Tokens.InputTokens
	sm.Tokens.OutputTokens += s.Tokens.OutputTokens
	sm.Tokens.CacheReadInputTokens += s.Tokens.CacheReadInputTokens
	sm.Tokens.CacheCreationInputTokens += s.Tokens.CacheCreationInputTokens
	sm.Tokens.CacheCreation5mTokens += s.Tokens.CacheCreation5mTokens
	sm.Tokens.CacheCreation1hTokens += s.Tokens.CacheCreation1hTokens
	sm.Tokens.TotalTokens += s.Tokens.TotalTokens
	sm.Tokens.ContextTokens += s.Tokens.ContextTokens

	if s.Kind == SpanKindTool {
		sm.ToolDurationMs += s.DurationMs
	} else if s.Kind == SpanKindGeneration {
		sm.GenerationDurationMs += s.DurationMs
	}

	if s.CostUSD > 0 {
		sm.TotalCostUSD += s.CostUSD
	}
	if s.Status == StatusError {
		sm.ErrorCount++
	}
}

// FinalizeRates computes ratios, multipliers, and percentages.
func (sm *SystemMetrics) FinalizeRates() {
	if sm.TotalTraces > 0 {
		sm.ErrorRate = (float64(sm.ErrorCount) / float64(sm.TotalTraces)) * 100.0
	}
	totalPrompt := sm.Tokens.InputTokens + sm.Tokens.CacheReadInputTokens
	if totalPrompt > 0 {
		sm.CacheHitRate = (float64(sm.Tokens.CacheReadInputTokens) / float64(totalPrompt)) * 100.0
	}
	sm.TotalUncachedCostUSD = sm.TotalCostUSD + sm.TotalCacheSavingsUSD
	if sm.TotalCostUSD > 0 && sm.TotalUncachedCostUSD > 0 {
		sm.CacheMultiplier = sm.TotalUncachedCostUSD / sm.TotalCostUSD
	}
	sm.Tokens.ContextWindowLimit = 200000
	if sm.TotalTraces > 0 {
		avgCtx := float64(sm.Tokens.ContextTokens) / float64(sm.TotalTraces)
		sm.AvgContextSaturation = (avgCtx / 200000.0) * 100.0
	}
}

