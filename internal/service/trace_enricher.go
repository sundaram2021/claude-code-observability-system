package service

import (
	"github.com/claude-code/observability/internal/domain"
)

// enrichTracePrompt discovers user prompt and assistant output from spans.
func enrichTracePrompt(t *domain.Trace) {
	if t != nil {
		t.EnrichPromptAndOutput()
	}
}

// enrichSpans computes costs and durations for individual spans.
func enrichSpans(t *domain.Trace) {
	for _, span := range t.Spans {
		span.CalculateDuration()
		if span.Tokens.TotalTokens == 0 {
			span.Tokens.TotalTokens = span.Tokens.InputTokens + span.Tokens.OutputTokens +
				span.Tokens.CacheReadInputTokens + span.Tokens.CacheCreationInputTokens
		}
		if span.CostUSD == 0 && span.Model != "" {
			cost, _ := domain.CalculateCost(span.Model, span.Tokens)
			span.CostUSD = cost
		}
		if span.Status == "" {
			span.Status = domain.StatusSuccess
		}
	}
}

// calculateTotals aggregates tokens, costs, performance, and overall status for the trace.
func calculateTotals(t *domain.Trace) {
	var totalCost float64
	var usage domain.TokenUsage
	var toolDur, genDur, sumTTFT, sumVel float64
	var toolCount, genCount int
	hasError, hadError, selfCorrected := false, false, false
	primaryModel := ""

	for _, span := range t.Spans {
		totalCost += span.CostUSD
		usage.InputTokens += span.Tokens.InputTokens
		usage.OutputTokens += span.Tokens.OutputTokens
		usage.CacheReadInputTokens += span.Tokens.CacheReadInputTokens
		usage.CacheCreationInputTokens += span.Tokens.CacheCreationInputTokens
		usage.CacheCreation5mTokens += span.Tokens.CacheCreation5mTokens
		usage.CacheCreation1hTokens += span.Tokens.CacheCreation1hTokens
		usage.TotalTokens += span.Tokens.TotalTokens

		if span.Model != "" && primaryModel == "" {
			primaryModel = span.Model
		}

		if span.Kind == domain.SpanKindTool {
			toolCount++
			toolDur += span.DurationMs
		} else if span.Kind == domain.SpanKindGeneration {
			genCount++
			genDur += span.DurationMs
			if span.TTFTMs > 0 {
				sumTTFT += span.TTFTMs
			}
			if span.TokensPerSec > 0 {
				sumVel += span.TokensPerSec
			}
		}

		if span.Status == domain.StatusError {
			hasError = true
			hadError = true
		} else if hadError && span.Status == domain.StatusSuccess {
			selfCorrected = true
		}
	}

	usage.ContextTokens = usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
	usage.ContextWindowLimit = 200000
	if usage.ContextTokens > 0 {
		usage.ContextSaturationPct = (float64(usage.ContextTokens) / 200000.0) * 100.0
	}

	t.TotalCost = totalCost
	t.TotalTokens = usage
	t.ToolCount = toolCount
	t.ToolDurationMs = toolDur
	t.GenerationDurationMs = genDur
	t.SelfCorrected = selfCorrected
	t.ContextSaturationPct = usage.ContextSaturationPct

	if genCount > 0 {
		t.AvgTTFTMs = sumTTFT / float64(genCount)
		t.TokensPerSec = sumVel / float64(genCount)
	}

	if primaryModel != "" {
		_, savings := domain.CalculateCost(primaryModel, usage)
		t.CacheSavingsUSD = savings
		uncached := totalCost + savings
		if totalCost > 0 && uncached > 0 {
			t.CacheMultiplier = uncached / totalCost
		}
	}

	if hasError {
		t.Status = domain.StatusError
	} else if t.Status == "" {
		t.Status = domain.StatusSuccess
	}
}
