package service

import (
	"sort"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

// MetricsService calculates high-level telemetry and financial metrics.
type MetricsService struct {
	traceRepo repository.TraceRepository
	sessRepo  repository.SessionRepository
}

// NewMetricsService creates an instance of MetricsService.
func NewMetricsService(traceRepo repository.TraceRepository, sessRepo repository.SessionRepository) *MetricsService {
	return &MetricsService{
		traceRepo: traceRepo,
		sessRepo:  sessRepo,
	}
}

// GetMetrics aggregates all system statistics across all recorded traces.
func (s *MetricsService) GetMetrics() (*domain.SystemMetrics, error) {
	traces, _, err := s.traceRepo.ListTraces(domain.TraceFilter{Limit: 5000})
	if err != nil {
		return nil, err
	}

	metrics := &domain.SystemMetrics{
		TotalTraces: len(traces),
	}

	toolMap := make(map[string]*domain.ToolUsageStat)
	modelMap := make(map[string]*domain.ModelUsageStat)
	durations := make([]float64, 0, len(traces))
	var totalLatency, sumTTFT, sumVel, totalSavings float64
	var countTTFT, countVel, errTraces, selfCorrectedCount int

	for _, t := range traces {
		durations = append(durations, t.DurationMs)
		totalLatency += t.DurationMs
		totalSavings += t.CacheSavingsUSD

		if t.AvgTTFTMs > 0 {
			sumTTFT += t.AvgTTFTMs
			countTTFT++
		}
		if t.TokensPerSec > 0 {
			sumVel += t.TokensPerSec
			countVel++
		}
		if t.Status == domain.StatusError || t.SelfCorrected {
			errTraces++
			if t.SelfCorrected {
				selfCorrectedCount++
			}
		}

		for _, span := range t.Spans {
			metrics.AddSpan(span)
			s.aggregateTool(span, toolMap)
			s.aggregateModel(span, modelMap)
		}
	}

	metrics.TotalCacheSavingsUSD = totalSavings
	if countTTFT > 0 {
		metrics.AvgTTFTMs = sumTTFT / float64(countTTFT)
	}
	if countVel > 0 {
		metrics.AvgTokensPerSec = sumVel / float64(countVel)
	}
	if errTraces > 0 {
		metrics.SelfCorrectionRate = (float64(selfCorrectedCount) / float64(errTraces)) * 100.0
	}

	s.finalizeDurations(metrics, durations, totalLatency)
	metrics.Tools = s.flattenTools(toolMap)
	metrics.Models = s.flattenModels(modelMap)
	metrics.FinalizeRates()

	return metrics, nil
}

func (s *MetricsService) aggregateTool(span *domain.Span, tools map[string]*domain.ToolUsageStat) {
	if span.Tool == nil || span.Tool.Name == "" {
		return
	}
	name := span.Tool.Name
	stat, exists := tools[name]
	if !exists {
		stat = &domain.ToolUsageStat{Name: name}
		tools[name] = stat
	}
	stat.CallCount++
	stat.AvgDurationMs = (stat.AvgDurationMs*float64(stat.CallCount-1) + span.DurationMs) / float64(stat.CallCount)
	if span.Tool.IsError || span.Status == domain.StatusError {
		stat.ErrorCount++
	}
}

func (s *MetricsService) aggregateModel(span *domain.Span, models map[string]*domain.ModelUsageStat) {
	if span.Model == "" {
		return
	}
	m := span.Model
	stat, exists := models[m]
	if !exists {
		stat = &domain.ModelUsageStat{Model: m}
		models[m] = stat
	}
	stat.Calls++
	stat.CostUSD += span.CostUSD
	stat.TotalTokens += span.Tokens.TotalTokens
}

func (s *MetricsService) finalizeDurations(m *domain.SystemMetrics, durs []float64, total float64) {
	if len(durs) == 0 {
		return
	}
	m.AvgLatencyMs = total / float64(len(durs))
	sort.Float64s(durs)
	p95Idx := int(float64(len(durs)) * 0.95)
	if p95Idx >= len(durs) {
		p95Idx = len(durs) - 1
	}
	m.P95LatencyMs = durs[p95Idx]
}

func (s *MetricsService) flattenTools(m map[string]*domain.ToolUsageStat) []domain.ToolUsageStat {
	res := make([]domain.ToolUsageStat, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	return res
}

func (s *MetricsService) flattenModels(m map[string]*domain.ModelUsageStat) []domain.ModelUsageStat {
	res := make([]domain.ModelUsageStat, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	return res
}

