package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

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

	loaded := 0
	for _, t := range traces {
		if t != nil && t.ID != "" {
			if err := store.SaveTrace(t); err == nil {
				loaded++
			}
		}
	}
	return loaded, nil
}
