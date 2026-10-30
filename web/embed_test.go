package web_test

import (
	"io"
	"testing"

	"github.com/claude-code/observability/web"
)

func TestGetFileSystem(t *testing.T) {
	fs := web.GetFileSystem()
	if fs == nil {
		t.Fatalf("expected non-nil http.FileSystem")
	}

	file, err := fs.Open("index.html")
	if err != nil {
		t.Fatalf("failed to open embedded index.html: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}

	if len(content) == 0 {
		t.Errorf("expected non-empty index.html")
	}
}
