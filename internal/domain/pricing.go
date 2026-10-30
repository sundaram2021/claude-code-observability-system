package domain

import "strings"

// ModelRates defines pricing per million tokens for a model.
type ModelRates struct {
	InputPerMillion      float64
	OutputPerMillion     float64
	CacheWritePerMillion float64
	CacheReadPerMillion  float64
}

// ModelRegistry maps model identifiers to their pricing tiers.
var ModelRegistry = map[string]ModelRates{
	"claude-3-7-sonnet": {
		InputPerMillion:      3.00,
		OutputPerMillion:     15.00,
		CacheWritePerMillion: 3.75,
		CacheReadPerMillion:  0.30,
	},
	"claude-3-5-sonnet": {
		InputPerMillion:      3.00,
		OutputPerMillion:     15.00,
		CacheWritePerMillion: 3.75,
		CacheReadPerMillion:  0.30,
	},
	"claude-3-5-haiku": {
		InputPerMillion:      0.80,
		OutputPerMillion:     4.00,
		CacheWritePerMillion: 1.00,
		CacheReadPerMillion:  0.08,
	},
	"claude-haiku-4-5": {
		InputPerMillion:      0.80,
		OutputPerMillion:     4.00,
		CacheWritePerMillion: 1.00,
		CacheReadPerMillion:  0.08,
	},
	"claude-haiku-5-5": {
		InputPerMillion:      0.80,
		OutputPerMillion:     4.00,
		CacheWritePerMillion: 1.00,
		CacheReadPerMillion:  0.08,
	},
	"haiku": {
		InputPerMillion:      0.80,
		OutputPerMillion:     4.00,
		CacheWritePerMillion: 1.00,
		CacheReadPerMillion:  0.08,
	},
	"claude-3-opus": {
		InputPerMillion:      15.00,
		OutputPerMillion:     75.00,
		CacheWritePerMillion: 18.75,
		CacheReadPerMillion:  1.50,
	},
}

var DefaultRates = ModelRates{
	InputPerMillion:      3.00,
	OutputPerMillion:     15.00,
	CacheWritePerMillion: 3.75,
	CacheReadPerMillion:  0.30,
}

// ResolveRates finds the matching pricing rates by fuzzy prefix matching.
func ResolveRates(model string) ModelRates {
	clean := strings.ToLower(strings.TrimSpace(model))
	for key, rates := range ModelRegistry {
		if strings.Contains(clean, key) {
			return rates
		}
	}
	return DefaultRates
}

// CalculateCost computes total cost and prompt cache savings in USD.
func CalculateCost(model string, u TokenUsage) (cost float64, savings float64) {
	rates := ResolveRates(model)
	const mtok = 1_000_000.0

	inputCost := (float64(u.InputTokens) * rates.InputPerMillion) / mtok
	outputCost := (float64(u.OutputTokens) * rates.OutputPerMillion) / mtok
	cacheWriteCost := (float64(u.CacheCreationInputTokens) * rates.CacheWritePerMillion) / mtok
	cacheReadCost := (float64(u.CacheReadInputTokens) * rates.CacheReadPerMillion) / mtok

	cost = inputCost + outputCost + cacheWriteCost + cacheReadCost

	// Savings is difference between full input price and discounted cache read price
	standardCostIfNoCache := (float64(u.CacheReadInputTokens) * rates.InputPerMillion) / mtok
	if standardCostIfNoCache > cacheReadCost {
		savings = standardCostIfNoCache - cacheReadCost
	}
	return cost, savings
}

