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
