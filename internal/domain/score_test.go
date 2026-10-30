package domain_test

import (
	"net/url"
	"testing"

	"github.com/claude-code/observability/internal/domain"
)

func TestScore_CreationAndTraceAttachment(t *testing.T) {
	score := domain.NewScore("sc_1", "trc_100", "sp_1", "accuracy", 0.95, "great output", "")
	if score.Source != "USER" {
		t.Errorf("expected default source USER, got %s", score.Source)
	}
	if score.Value != 0.95 || score.Name != "accuracy" {
		t.Errorf("unexpected score values: %+v", score)
	}

	trace := &domain.Trace{ID: "trc_100"}
	trace.AddScore(score)
	if len(trace.Scores) != 1 || trace.Scores[0].ID != "sc_1" {
		t.Fatalf("failed to attach score to trace")
	}

	// Nil score safety
	trace.AddScore(nil)
	if len(trace.Scores) != 1 {
		t.Errorf("expected nil score to be ignored")
	}
}

func TestFilter_TagsAndUser(t *testing.T) {
	q := url.Values{}
	q.Set("tag", "prod")
	q.Set("userId", "usr_dev_1")

	f := domain.ParseTraceFilter(q)
	if f.Tag != "prod" || f.UserID != "usr_dev_1" {
		t.Errorf("unexpected parsed tag or userId: %+v", f)
	}

	tMatch := &domain.Trace{
		Tags:   []string{"v1", "PROD"},
		UserID: "usr_dev_1",
	}
	if !f.Matches(tMatch) {
		t.Errorf("expected case-insensitive tag and user match")
	}

	tMismatchUser := &domain.Trace{Tags: []string{"prod"}, UserID: "other"}
	if f.Matches(tMismatchUser) {
		t.Errorf("expected mismatch on userId")
	}

	tMismatchTag := &domain.Trace{Tags: []string{"staging"}, UserID: "usr_dev_1"}
	if f.Matches(tMismatchTag) {
		t.Errorf("expected mismatch on tag")
	}
}

