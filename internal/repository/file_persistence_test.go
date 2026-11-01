package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

func TestFilePersistence_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "traces_test.json")

	store := NewMemoryStore(100, 50)
	tr := &domain.Trace{
		ID:        "trc_file_1",
		SessionID: "sess_file_1",
		Project:   "test-proj",
		Name:      "test-trace",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
	}
	_ = store.SaveTrace(tr)

	err := SaveToFile(store, filePath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("File does not exist: %s", filePath)
	}

	newStore := NewMemoryStore(100, 50)
	loaded, err := LoadFromFile(newStore, filePath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if loaded != 1 {
		t.Errorf("expected 1 trace loaded, got %d", loaded)
	}

	loadedTrace, err := newStore.GetTrace("trc_file_1")
	if err != nil || loadedTrace == nil {
		t.Fatalf("GetTrace failed: %v", err)
	}
	if loadedTrace.Name != "test-trace" {
		t.Errorf("expected trace name 'test-trace', got '%s'", loadedTrace.Name)
	}
}

func TestFilePersistence_EdgeCases(t *testing.T) {
	// Nil store
	if err := SaveToFile(nil, ""); err != ErrInvalid {
		t.Errorf("expected ErrInvalid for nil store, got %v", err)
	}
	if _, err := LoadFromFile(nil, ""); err != ErrInvalid {
		t.Errorf("expected ErrInvalid for nil store, got %v", err)
	}

	// Non-existent file
	store := NewMemoryStore(10, 10)
	loaded, err := LoadFromFile(store, filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil || loaded != 0 {
		t.Errorf("expected 0 loaded without error for nonexistent file, got %d, %v", loaded, err)
	}

	// Corrupt file
	corruptFile := filepath.Join(t.TempDir(), "corrupt.json")
	_ = os.WriteFile(corruptFile, []byte("{bad json"), 0644)
	if _, err := LoadFromFile(store, corruptFile); err == nil {
		t.Errorf("expected error loading corrupt JSON")
	}

	// Default path test
	defPath := DefaultTraceFilePath()
	if defPath == "" {
		t.Errorf("expected non-empty default trace file path")
	}
}

func TestFilePersistence_LoadEnrichesPrompts(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "prompt_traces.json")

	store1 := NewMemoryStore(10, 10)
	// Trace 1: generic name, has user prompt in metadata
	tr1 := &domain.Trace{
		ID:        "trc_p1",
		SessionID: "sess_p1",
		Name:      "Claude Code Turn",
		Spans: []*domain.Span{
			{
				ID:   "sp_1",
				Name: "Event: user_prompt",
				Metadata: map[string]string{
					"prompt": "explain codebase",
				},
			},
		},
	}
	// Trace 2: in same session, but no prompt metadata
	tr2 := &domain.Trace{
		ID:        "trc_p2",
		SessionID: "sess_p1",
		Name:      "claude_code.llm_request",
		Spans: []*domain.Span{
			{ID: "sp_2", Name: "claude_code.llm_request"},
		},
	}
	_ = store1.SaveTrace(tr1)
	_ = store1.SaveTrace(tr2)
	if err := SaveToFile(store1, filePath); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	store2 := NewMemoryStore(10, 10)
	loaded, err := LoadFromFile(store2, filePath)
	if err != nil || loaded != 1 {
		t.Fatalf("LoadFromFile failed: %v, loaded=%d", err, loaded)
	}

	list, _, _ := store2.ListTraces(domain.TraceFilter{})
	if len(list) != 1 {
		t.Fatalf("expected 1 merged trace in list, got %d", len(list))
	}
	l1 := list[0]
	if l1.UserPrompt != "explain codebase" || l1.Name != "explain codebase" {
		t.Errorf("expected merged trace with prompt 'explain codebase', got %+v", l1)
	}
	if len(l1.Spans) != 2 {
		t.Errorf("expected 2 merged spans in l1, got %d", len(l1.Spans))
	}
}

func TestFilePersistence_IsCorrelatedBranches(t *testing.T) {
	now := time.Now()
	// Different session IDs
	t1 := &domain.Trace{SessionID: "s1"}
	t2 := &domain.Trace{SessionID: "s2"}
	if isCorrelated(t1, t2) {
		t.Errorf("expected false for different session IDs")
	}

	// Different non-empty prompts
	t1 = &domain.Trace{SessionID: "s1", UserPrompt: "prompt A"}
	t2 = &domain.Trace{SessionID: "s1", UserPrompt: "prompt B"}
	if isCorrelated(t1, t2) {
		t.Errorf("expected false for different prompts")
	}

	// Zero start time
	t1 = &domain.Trace{SessionID: "s1"}
	t2 = &domain.Trace{SessionID: "s1", StartTime: now}
	if !isCorrelated(t1, t2) {
		t.Errorf("expected true when start time is zero")
	}

	// Beyond 5 minutes and no overlap
	t1 = &domain.Trace{SessionID: "s1", StartTime: now, EndTime: now.Add(time.Second)}
	t2 = &domain.Trace{SessionID: "s1", StartTime: now.Add(10 * time.Minute), EndTime: now.Add(11 * time.Minute)}
	if isCorrelated(t1, t2) {
		t.Errorf("expected false for traces 10 minutes apart")
	}

	// Overlapping interval
	t1 = &domain.Trace{SessionID: "s1", StartTime: now, EndTime: now.Add(10 * time.Minute)}
	t2 = &domain.Trace{SessionID: "s1", StartTime: now.Add(6 * time.Minute), EndTime: now.Add(12 * time.Minute)}
	if !isCorrelated(t1, t2) {
		t.Errorf("expected true for overlapping traces")
	}
}


