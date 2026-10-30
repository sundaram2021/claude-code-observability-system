package agent_test

import (
	"context"
	"testing"

	"github.com/claude-code/observability/internal/agent"
	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestDefaultTools(t *testing.T) {
	tools := agent.DefaultTools()
	if len(tools) != 4 {
		t.Fatalf("expected 4 default tools, got %d", len(tools))
	}
	expected := map[string]bool{"bash": true, "file_edit": true, "glob": true, "view": true}
	for _, tool := range tools {
		if !expected[tool.Name] {
			t.Errorf("unexpected tool name: %s", tool.Name)
		}
		if tool.Description == "" || tool.InputSchema == nil {
			t.Errorf("missing description or schema for tool %s", tool.Name)
		}
	}
}

func TestExecuteTool(t *testing.T) {
	// 1. Bash test command
	res1 := agent.ExecuteTool("bash", `{"command": "go test ./..."}`)
	if res1.ToolName != "bash" || res1.ExitCode != 0 || res1.IsError {
		t.Errorf("unexpected bash test result: %+v", res1)
	}

	// 2. Bash git status command
	res2 := agent.ExecuteTool("bash", `{"command": "git status"}`)
	if res2.ExitCode != 0 {
		t.Errorf("unexpected git status result: %+v", res2)
	}

	// 3. Bash generic command
	res3 := agent.ExecuteTool("bash", `{"command": "echo hello"}`)
	if res3.ExitCode != 0 {
		t.Errorf("unexpected bash generic result: %+v", res3)
	}

	// 4. File edit tool
	res4 := agent.ExecuteTool("file_edit", `{"file_path": "main.go", "target": "a", "replacement": "b"}`)
	if res4.ToolName != "file_edit" || res4.ExitCode != 0 {
		t.Errorf("unexpected file_edit result: %+v", res4)
	}

	// 5. Glob tool
	res5 := agent.ExecuteTool("glob", `{"pattern": "*.go"}`)
	if res5.ToolName != "glob" {
		t.Errorf("unexpected glob result: %+v", res5)
	}

	// 6. View tool
	res6 := agent.ExecuteTool("view", `{"file_path": "main.go"}`)
	if res6.ToolName != "view" {
		t.Errorf("unexpected view result: %+v", res6)
	}

	// 7. Unknown/default tool
	res7 := agent.ExecuteTool("custom_tool", `{"action": "test"}`)
	if res7.ToolName != "custom_tool" {
		t.Errorf("unexpected custom_tool result: %+v", res7)
	}

	// 8. Malformed JSON input
	res8 := agent.ExecuteTool("bash", `invalid-json`)
	if res8.ToolName != "bash" {
		t.Errorf("expected handling of invalid json")
	}
}

func TestRunner_Run(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	runner := agent.NewRunner(traceSvc, "", "")

	ctx := context.Background()
	req := agent.RunRequest{
		Prompt:   "Fix unit tests and refactor tools",
		Simulate: true,
	}

	resp, err := runner.Run(ctx, req)
	if err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected runner success")
	}
	if resp.TraceID == "" || resp.SessionID == "" {
		t.Errorf("missing traceId or sessionId in response")
	}
	if resp.Turns != 4 {
		t.Errorf("expected 4 turns, got %d", resp.Turns)
	}
	if resp.Tokens.TotalTokens <= 0 {
		t.Errorf("expected total tokens > 0, got %d", resp.Tokens.TotalTokens)
	}

	// Verify trace is stored in repo
	trace, err := traceSvc.GetTrace(resp.TraceID)
	if err != nil || trace == nil {
		t.Fatalf("trace not found in repository")
	}
	if trace.Status != domain.StatusSuccess {
		t.Errorf("expected trace status success")
	}
	if len(trace.Spans) != 5 { // root + 2 gens + 2 tools
		t.Errorf("expected 5 spans in trace, got %d", len(trace.Spans))
	}
}

