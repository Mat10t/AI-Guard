package limits

import (
	"testing"
	"time"
)

func TestBucket(t *testing.T) {
	tm := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)

	day, err := Bucket(tm, "day")
	if err != nil || day != "2026-04-09" {
		t.Fatalf("day bucket mismatch: %s, err=%v", day, err)
	}

	week, err := Bucket(tm, "week")
	if err != nil || week == "" {
		t.Fatalf("week bucket error: %v", err)
	}

	month, err := Bucket(tm, "month")
	if err != nil || month != "2026-04" {
		t.Fatalf("month bucket mismatch: %s, err=%v", month, err)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("expected 0 got %d", got)
	}
	if got := EstimateTokens("abcd"); got != 1 {
		t.Fatalf("expected 1 got %d", got)
	}
	if got := EstimateTokens("abcdefghijkl"); got != 3 {
		t.Fatalf("expected 3 got %d", got)
	}
}
