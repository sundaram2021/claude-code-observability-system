package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

var fileMu sync.Mutex

// DefaultTraceFilePath returns the default path for persisting traces: ~/.claude/observability_traces.json
func DefaultTraceFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "observability_traces.json"
	}
	return filepath.Join(home, ".claude", "observability_traces.json")
}

// SaveToFile writes all traces from the store to the specified file path atomically.
func SaveToFile(store TraceRepository, filePath string) error {
	if store == nil {
		return ErrInvalid
	}
	if filePath == "" {
		filePath = DefaultTraceFilePath()
	}

	traces, _, err := store.ListTraces(domain.TraceFilter{Limit: 1000})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(traces, "", "  ")
	if err != nil {
		return err
	}

	fileMu.Lock()
	defer fileMu.Unlock()

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, filePath)
}

// LoadFromFile reads traces from the file path and populates the store.
func LoadFromFile(store TraceRepository, filePath string) (int, error) {
	if store == nil {
		return 0, ErrInvalid
	}
	if filePath == "" {
		filePath = DefaultTraceFilePath()
	}

	fileMu.Lock()
	defer fileMu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var traces []*domain.Trace
	if err := json.Unmarshal(data, &traces); err != nil {
		return 0, err
	}

	merged := make([]*domain.Trace, 0, len(traces))
	for _, t := range traces {
		if t == nil || t.ID == "" || t.IsStartupPlumbing() {
			continue
		}
		enrichFromSpans(t)
		var existing *domain.Trace
		for _, m := range merged {
			if m.ID == t.ID || (m.SessionID != "" && m.SessionID == t.SessionID && isCorrelated(m, t)) {
				existing = m
				break
			}
		}
		if existing != nil {
			existing.MergeFrom(t)
		} else {
			merged = append(merged, t)
		}
	}

	loaded := 0
	for _, t := range merged {
		if t.UserPrompt != "" && (t.Name == "" || t.Name == "Claude Code Turn" || strings.HasPrefix(t.Name, "claude_code.")) {
			t.Name = t.UserPrompt
		}
		if err := store.SaveTrace(t); err == nil {
			loaded++
		}
	}
	return loaded, nil
}

func enrichFromSpans(t *domain.Trace) {
	if t.UserPrompt == "" {
		for _, sp := range t.Spans {
			if sp != nil && sp.Metadata != nil {
				for _, k := range []string{"user_prompt", "prompt", "gen_ai.prompt", "query"} {
					if val := sp.Metadata[k]; val != "" {
						t.UserPrompt = val
						break
					}
				}
			}
			if t.UserPrompt != "" {
				break
			}
		}
	}
	if t.UserOutput == "" {
		for _, sp := range t.Spans {
			if sp != nil && sp.Metadata != nil {
				for _, k := range []string{"response", "assistant_response", "text", "content"} {
					if val := sp.Metadata[k]; val != "" && !strings.HasPrefix(val, "claude_code.") {
						t.UserOutput = val
						break
					}
				}
			}
			if t.UserOutput != "" {
				break
			}
		}
	}
}

func isCorrelated(a, b *domain.Trace) bool {
	if a.UserPrompt != "" && b.UserPrompt != "" && a.UserPrompt == b.UserPrompt {
		return true
	}
	spanIDs := make(map[string]bool, len(a.Spans))
	for _, s := range a.Spans {
		spanIDs[s.ID] = true
	}
	for _, s := range b.Spans {
		if spanIDs[s.ID] {
			return true
		}
	}
	diff := a.StartTime.Sub(b.StartTime)
	if diff < 0 {
		diff = -diff
	}
	return diff < 15*time.Second && (a.UserPrompt == "" || b.UserPrompt == "" || a.UserPrompt == b.UserPrompt)
}

