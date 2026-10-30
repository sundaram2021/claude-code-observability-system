package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTools returns Claude Code standard tool schemas.
func DefaultTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "bash",
			Description: "Run shell command in the workspace directory",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string"},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "file_edit",
			Description: "Replace target text with new content in a specified file",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path":   map[string]any{"type": "string"},
					"target":      map[string]any{"type": "string"},
					"replacement": map[string]any{"type": "string"},
				},
				"required": []string{"file_path", "target", "replacement"},
			},
		},
		{
			Name:        "glob",
			Description: "Fast file pattern matching",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{"type": "string"},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "view",
			Description: "View file contents within an optional line range",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path":  map[string]any{"type": "string"},
					"start_line": map[string]any{"type": "integer"},
					"end_line":   map[string]any{"type": "integer"},
				},
				"required": []string{"file_path"},
			},
		},
	}
}

// ExecuteTool simulates or executes a tool invocation.
func ExecuteTool(name string, inputJSON string) ToolExecResult {
	start := time.Now()
	_ = start // duration tracked by span

	var inputMap map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &inputMap); err != nil {
		inputMap = map[string]any{"raw": inputJSON}
	}

	switch name {
	case "bash":
		cmd, _ := inputMap["command"].(string)
		return executeBash(cmd, inputJSON)
	case "file_edit":
		path, _ := inputMap["file_path"].(string)
		return ToolExecResult{
			ToolName: name,
			Input:    inputJSON,
			Output:   fmt.Sprintf("Successfully edited %s (0 lines added, 0 lines deleted)", path),
			ExitCode: 0,
		}
	case "glob":
		pattern, _ := inputMap["pattern"].(string)
		matches, _ := filepath.Glob(pattern)
		return ToolExecResult{
			ToolName: name,
			Input:    inputJSON,
			Output:   fmt.Sprintf("Matched %d files:\n%s", len(matches), strings.Join(matches, "\n")),
			ExitCode: 0,
		}
	case "view":
		path, _ := inputMap["file_path"].(string)
		return ToolExecResult{
			ToolName: name,
			Input:    inputJSON,
			Output:   fmt.Sprintf("Contents of %s (lines 1-45 displayed)", path),
			ExitCode: 0,
		}
	default:
		return ToolExecResult{
			ToolName: name,
			Input:    inputJSON,
			Output:   fmt.Sprintf("Tool %s executed successfully", name),
			ExitCode: 0,
		}
	}
}

func executeBash(cmd, inputJSON string) ToolExecResult {
	if strings.Contains(cmd, "test") {
		return ToolExecResult{
			ToolName: "bash",
			Input:    inputJSON,
			Output:   "PASS\nok   github.com/claude-code/observability 0.042s\nALL TESTS PASSED",
			ExitCode: 0,
		}
	}
	if strings.Contains(cmd, "git status") {
		return ToolExecResult{
			ToolName: "bash",
			Input:    inputJSON,
			Output:   "On branch main\nChanges not staged for commit:\n  modified: internal/agent/runner.go",
			ExitCode: 0,
		}
	}
	return ToolExecResult{
		ToolName: "bash",
		Input:    inputJSON,
		Output:   fmt.Sprintf("$ %s\nexit status 0", cmd),
		ExitCode: 0,
	}
}

