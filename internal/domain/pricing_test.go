package domain_test

import (
	"testing"

	"github.com/claude-code/observability/internal/domain"
)

func TestPricingResolution(t *testing.T) {
	tests := []struct {
		model    string
		expected float64
	}{
		{"claude-3-7-sonnet-20250219", 3.00},
		{"claude-3-5-sonnet-latest", 3.00},
		{"claude-3-5-haiku-20241022", 0.80},
		{"claude-3-opus-20240229", 15.00},
		{"unknown-model", 3.00}, // default fallback
	}

	for _, tc := range tests {
		rates := domain.ResolveRates(tc.model)
		if rates.InputPerMillion != tc.expected {
			t.Errorf("model %s: expected input rate %.2f, got %.2f", tc.model, tc.expected, rates.InputPerMillion)
		}
	}
}

func TestCalculateCostWithCacheSavings(t *testing.T) {
	usage := domain.TokenUsage{
		InputTokens:              1000,
		OutputTokens:             500,
		CacheReadInputTokens:     10000,
		CacheCreationInputTokens: 2000,
	}

	cost, savings := domain.CalculateCost("claude-3-7-sonnet", usage)

	// Expected calculations for 3.7 Sonnet:
	// input: 1,000 * 3.00 / 1e6 = 0.003
	// output: 500 * 15.00 / 1e6 = 0.0075
	// cache write: 2,000 * 3.75 / 1e6 = 0.0075
	// cache read: 10,000 * 0.30 / 1e6 = 0.003
	// total = 0.021
	// savings: 10,000 * (3.00 - 0.30) / 1e6 = 0.027
	if cost <= 0 {
		t.Fatalf("expected cost > 0, got %f", cost)
	}
	if savings <= 0 {
		t.Fatalf("expected cache savings > 0, got %f", savings)
	}

	expectedCost := 0.021
	expectedSavings := 0.027
	const epsilon = 0.0001

	if diff := cost - expectedCost; diff > epsilon || diff < -epsilon {
		t.Errorf("expected cost ~%.4f, got %.4f", expectedCost, cost)
	}
	if diff := savings - expectedSavings; diff > epsilon || diff < -epsilon {
		t.Errorf("expected savings ~%.4f, got %.4f", expectedSavings, savings)
	}
}

