package main

import (
	"math"
	"testing"
	"time"
)

func TestPeriodTTL(t *testing.T) {
	if got := periodTTL("day"); got != 24*time.Hour {
		t.Fatalf("unexpected day ttl: %v", got)
	}
	if got := periodTTL("week"); got != 7*24*time.Hour {
		t.Fatalf("unexpected week ttl: %v", got)
	}
	if got := periodTTL("month"); got != 31*24*time.Hour {
		t.Fatalf("unexpected month ttl: %v", got)
	}
}

func TestBudgetTokenConversions(t *testing.T) {
	usdPerToken := blendedUSDPerToken(0.00015, 0.00060)
	if usdPerToken <= 0 {
		t.Fatalf("expected positive usdPerToken")
	}

	budget := computeBudgetFromTokens(2000, usdPerToken)
	if budget <= 0 {
		t.Fatalf("expected positive budget")
	}

	tokens := computeTokensFromBudget(budget, usdPerToken)
	if tokens != 2000 {
		t.Fatalf("expected 2000 tokens from budget, got %d", tokens)
	}
}

func TestComputeTokensFromBudgetFloor(t *testing.T) {
	usdPerToken := blendedUSDPerToken(0.00015, 0.00060)
	got := computeTokensFromBudget(usdPerToken*2.99, usdPerToken)
	if got != 2 {
		t.Fatalf("expected floor conversion to 2, got %d", got)
	}
}

func TestZeroPriceBudgetConversion(t *testing.T) {
	if got := computeTokensFromBudget(1.0, 0); got != 0 {
		t.Fatalf("expected 0 tokens for zero usdPerToken, got %d", got)
	}
	if got := computeBudgetFromTokens(10, 0); got != 0 {
		t.Fatalf("expected 0 budget for zero usdPerToken, got %f", got)
	}
}

func TestRoundBudgetUSD(t *testing.T) {
	got := roundBudgetUSD(0.123456789123499)
	want := 0.123456789123
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("unexpected rounded value: got=%0.15f want=%0.15f", got, want)
	}
}

func TestNormalizeSyncSource(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "tokens"},
		{in: "tokens", want: "tokens"},
		{in: "TOKENS", want: "tokens"},
		{in: "budget", want: "budget"},
		{in: "BUDGET", want: "budget"},
		{in: "other", want: "tokens"},
	}

	for _, tc := range cases {
		got := normalizeSyncSource(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeSyncSource(%q): got=%q want=%q", tc.in, got, tc.want)
		}
	}
}
