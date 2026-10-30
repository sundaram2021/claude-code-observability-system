package repository

import (
	"sync"
)

// RingBuffer provides a thread-safe, bounded, zero-leak circular queue.
type RingBuffer[T any] struct {
	mu       sync.RWMutex
	elements []T
	capacity int
	head     int
	size     int
	onEvict  func(evicted T)
}

// NewRingBuffer allocates a new RingBuffer with a fixed capacity.
func NewRingBuffer[T any](capacity int, onEvict func(T)) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 1000
	}
	return &RingBuffer[T]{
		elements: make([]T, capacity),
		capacity: capacity,
		onEvict:  onEvict,
	}
}

// Push adds an item to the ring buffer. If capacity is exceeded, oldest is evicted.
func (r *RingBuffer[T]) Push(item T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == r.capacity {
		oldestIdx := (r.head - r.size + r.capacity) % r.capacity
		if r.onEvict != nil {
			r.onEvict(r.elements[oldestIdx])
		}
		var zero T
		r.elements[oldestIdx] = zero // clear memory reference to prevent leaks
	} else {
		r.size++
	}

	r.elements[r.head] = item
	r.head = (r.head + 1) % r.capacity
}

// All returns a slice of all items ordered from newest to oldest.
func (r *RingBuffer[T]) All() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]T, r.size)
	for i := 0; i < r.size; i++ {
		idx := (r.head - 1 - i + r.capacity) % r.capacity
		result[i] = r.elements[idx]
	}
	return result
}

// Size returns the current number of elements in the buffer.
func (r *RingBuffer[T]) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}

// Clear resets the ring buffer and frees stored references.
func (r *RingBuffer[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var zero T
	for i := range r.elements {
		r.elements[i] = zero
	}
	r.head = 0
	r.size = 0
}

