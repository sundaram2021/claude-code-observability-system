package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// BuildSpanTree organizes a flat slice of spans into a hierarchical tree.
func BuildSpanTree(spans []*domain.Span) *domain.Span {
	if len(spans) == 0 {
		return nil
	}

	spanMap := make(map[string]*domain.Span, len(spans))
	// Deep copy or reset children slices
	for _, s := range spans {
		s.Children = nil
		spanMap[s.ID] = s
	}

	var root *domain.Span
	for _, s := range spans {
		if s.IsRoot() {
			if root == nil {
				root = s
			}
		} else if parent, exists := spanMap[s.ParentID]; exists {
			parent.Children = append(parent.Children, s)
		} else {
			// If parent is missing, treat as fallback root
			if root == nil {
				root = s
			}
		}
	}

	// Fallback to first span if no explicit root found
	if root == nil && len(spans) > 0 {
		root = spans[0]
	}

	return root
}

// generateID creates a collision-resistant unique identifier with a given prefix.
func generateID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to nanosecond timestamp if entropy read fails
		return prefix + hex.EncodeToString([]byte(time.Now().String()))[:16]
	}
	return prefix + hex.EncodeToString(bytes)
}

