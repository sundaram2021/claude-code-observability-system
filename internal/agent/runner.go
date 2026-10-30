package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/service"
)

// Runner manages execution and instrumentation of Claude Code tasks.
type Runner struct {
	traceService *service.TraceService
	apiKey       string
	defaultModel string
}

// NewRunner creates a new Claude Code agent runner.
func NewRunner(traceService *service.TraceService, apiKey, defaultModel string) *Runner {
	if defaultModel == "" {
		defaultModel = "claude-3-7-sonnet"
	}
	return &Runner{
		traceService: traceService,
		apiKey:       apiKey,
		defaultModel: defaultModel,
	}
}

// Run executes or simulates a Claude Code agent task and returns trace outcomes.
func (r *Runner) Run(ctx context.Context, req RunRequest) (*RunResponse, error) {
	if req.Model == "" {
		req.Model = r.defaultModel
	}
	if req.SessionID == "" {
		req.SessionID = "sess_" + fmt.Sprintf("%d", time.Now().UnixNano())[8:]
	}
	if req.Project == "" {
		req.Project = "observability-system"
	}

	start := time.Now()
	traceID := "trc_" + fmt.Sprintf("%d", time.Now().UnixNano())[8:]

	trace := &domain.Trace{
		ID:        traceID,
		SessionID: req.SessionID,
		Project:   req.Project,
		Name:      "Agent: " + req.Prompt,
		StartTime: start,
		Spans:     make([]*domain.Span, 0, 8),
	}

	rootSpan := &domain.Span{
		ID:        "span_root_" + traceID,
		TraceID:   traceID,
		SessionID: req.SessionID,
		Name:      "claude.code.task",
		Kind:      domain.SpanKindWorkflow,
		Status:    domain.StatusSuccess,
		StartTime: start,
	}
	trace.Spans = append(trace.Spans, rootSpan)

	// Execute simulated Claude Code multi-turn loop
	turns, finalOutput := r.simulateTurns(trace, rootSpan.ID, req)

	end := time.Now()
	rootSpan.EndTime = end
	rootSpan.CalculateDuration()
	trace.EndTime = end
	trace.DurationMs = float64(end.Sub(start).Microseconds()) / 1000.0

	if err := r.traceService.IngestTrace(trace); err != nil {
		return nil, err
	}

	return &RunResponse{
		TraceID:    trace.ID,
		SessionID:  req.SessionID,
		Success:    trace.Status == domain.StatusSuccess,
		Output:     finalOutput,
		Turns:      turns,
		DurationMs: trace.DurationMs,
		CostUSD:    trace.TotalCost,
		Tokens:     trace.TotalTokens,
	}, nil
}

func (r *Runner) simulateTurns(trace *domain.Trace, rootID string, req RunRequest) (int, string) {
	now := time.Now()

	// Turn 1: LLM reasoning & planning
	gen1 := &domain.Span{
		ID:        "gen_1_" + trace.ID,
		TraceID:   trace.ID,
		ParentID:  rootID,
		SessionID: req.SessionID,
		Name:      "claude.generation:plan",
		Kind:      domain.SpanKindGeneration,
		Status:    domain.StatusSuccess,
		Model:     req.Model,
		StartTime: now,
		EndTime:   now.Add(450 * time.Millisecond),
		Tokens: domain.TokenUsage{
			InputTokens:              1200,
			OutputTokens:             350,
			CacheReadInputTokens:     3500,
			CacheCreationInputTokens: 0,
		},
	}
	trace.Spans = append(trace.Spans, gen1)

	// Turn 2: Bash test execution
	tool1Res := ExecuteTool("bash", `{"command": "go test ./..."}`)
	tool1 := &domain.Span{
		ID:        "tool_1_" + trace.ID,
		TraceID:   trace.ID,
		ParentID:  rootID,
		SessionID: req.SessionID,
		Name:      "tool:bash",
		Kind:      domain.SpanKindTool,
		Status:    domain.StatusSuccess,
		StartTime: now.Add(460 * time.Millisecond),
		EndTime:   now.Add(620 * time.Millisecond),
		Tool: &domain.ToolCall{
			Name:     tool1Res.ToolName,
			Input:    tool1Res.Input,
			Output:   tool1Res.Output,
			ExitCode: tool1Res.ExitCode,
		},
	}
	trace.Spans = append(trace.Spans, tool1)

	// Turn 3: FileEdit code modification
	tool2Res := ExecuteTool("file_edit", `{"file_path": "internal/domain/trace.go", "target": "old", "replacement": "new"}`)
	tool2 := &domain.Span{
		ID:        "tool_2_" + trace.ID,
		TraceID:   trace.ID,
		ParentID:  rootID,
		SessionID: req.SessionID,
		Name:      "tool:file_edit",
		Kind:      domain.SpanKindTool,
		Status:    domain.StatusSuccess,
		StartTime: now.Add(630 * time.Millisecond),
		EndTime:   now.Add(710 * time.Millisecond),
		Tool: &domain.ToolCall{
			Name:     tool2Res.ToolName,
			Input:    tool2Res.Input,
			Output:   tool2Res.Output,
			ExitCode: tool2Res.ExitCode,
		},
	}
	trace.Spans = append(trace.Spans, tool2)

	// Final LLM Turn: Summary
	gen2 := &domain.Span{
		ID:        "gen_2_" + trace.ID,
		TraceID:   trace.ID,
		ParentID:  rootID,
		SessionID: req.SessionID,
		Name:      "claude.generation:summary",
		Kind:      domain.SpanKindGeneration,
		Status:    domain.StatusSuccess,
		Model:     req.Model,
		StartTime: now.Add(720 * time.Millisecond),
		EndTime:   now.Add(980 * time.Millisecond),
		Tokens: domain.TokenUsage{
			InputTokens:              850,
			OutputTokens:             280,
			CacheReadInputTokens:     4800,
			CacheCreationInputTokens: 0,
		},
	}
	trace.Spans = append(trace.Spans, gen2)

	output := fmt.Sprintf("Claude Code completed task '%s': verified tests with bash and updated code via file_edit.", req.Prompt)
	return 4, output
}

