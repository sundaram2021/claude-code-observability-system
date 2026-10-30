package repository_test

import (
	"testing"

	"github.com/claude-code/observability/internal/repository"
)

func TestRingBuffer_PushAndAll(t *testing.T) {
	buf := repository.NewRingBuffer[string](3, nil)

	buf.Push("a")
	buf.Push("b")
	buf.Push("c")

	items := buf.All()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// Ordered newest to oldest
	if items[0] != "c" || items[1] != "b" || items[2] != "a" {
		t.Errorf("unexpected order: %v", items)
	}
}

func TestRingBuffer_Eviction(t *testing.T) {
	var evicted []string
	buf := repository.NewRingBuffer[string](3, func(item string) {
		evicted = append(evicted, item)
	})

	buf.Push("1")
	buf.Push("2")
	buf.Push("3")
	buf.Push("4") // should evict "1"
	buf.Push("5") // should evict "2"

	if len(evicted) != 2 {
		t.Fatalf("expected 2 evictions, got %d", len(evicted))
	}
	if evicted[0] != "1" || evicted[1] != "2" {
		t.Errorf("expected [1, 2], got %v", evicted)
	}

	all := buf.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(all))
	}
	if all[0] != "5" || all[1] != "4" || all[2] != "3" {
		t.Errorf("expected [5, 4, 3], got %v", all)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	buf := repository.NewRingBuffer[int](5, nil)
	buf.Push(10)
	buf.Push(20)

	if buf.Size() != 2 {
		t.Fatalf("expected size 2, got %d", buf.Size())
	}

	buf.Clear()
	if buf.Size() != 0 {
		t.Fatalf("expected size 0 after clear, got %d", buf.Size())
	}
	if len(buf.All()) != 0 {
		t.Fatalf("expected empty slice from All() after clear")
	}
}

