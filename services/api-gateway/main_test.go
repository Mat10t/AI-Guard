package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestParseOutputTokens(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":5,"completion_tokens":9,"total_tokens":14}}`)
	if got := parseOutputTokens(body); got != 9 {
		t.Fatalf("expected 9, got %d", got)
	}
}

func TestToInt(t *testing.T) {
	if toInt("10") != 10 {
		t.Fatalf("expected 10")
	}
	if toInt(3.0) != 3 {
		t.Fatalf("expected 3")
	}
	if toInt("bad") != 0 {
		t.Fatalf("expected 0")
	}
}

func TestEstimatePromptTokens(t *testing.T) {
	req := chatRequest{Messages: []chatMessage{{Role: "user", Content: "abcd"}, {Role: "user", Content: "abcdefgh"}}}
	if got := estimatePromptTokens(req); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestEstimatedCostFromRates(t *testing.T) {
	got := estimatedCostFromRates(0.00020, 0.00080, 1000, 500)
	want := 0.0006
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("unexpected cost: got=%f want=%f", got, want)
	}
}

func TestEstimatedCostByLookupFallbackToZero(t *testing.T) {
	if got := estimatedCostByLookup("unknown", 0, 0, 100, 100, pgx.ErrNoRows); got != 0 {
		t.Fatalf("expected 0 for unknown model, got=%f", got)
	}
	if got := estimatedCostByLookup("broken", 0, 0, 100, 100, errors.New("db error")); got != 0 {
		t.Fatalf("expected 0 for lookup error, got=%f", got)
	}
}

func TestExecuteWithResilienceUsesFallbackModel(t *testing.T) {
	primaryURL := "http://primary.test/v1/chat/completions"
	fallbackURL := "http://fallback.test/internal/mock/completions"
	primaryCalls := 0
	fallbackCalls := 0
	var seenModel string

	s := &service{
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.String() {
				case primaryURL:
					primaryCalls++
					return &http.Response{
						StatusCode: http.StatusBadGateway,
						Body:       io.NopCloser(strings.NewReader(`{"code":"provider_unavailable"}`)),
						Header:     make(http.Header),
					}, nil
				case fallbackURL:
					fallbackCalls++
					var payload chatRequest
					_ = json.NewDecoder(req.Body).Decode(&payload)
					seenModel = payload.Model
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"usage":{"completion_tokens":7}}`)),
						Header:     make(http.Header),
					}, nil
				default:
					return nil, errors.New("unexpected url: " + req.URL.String())
				}
			}),
		},
	}

	req := chatRequest{
		Model: "gpt-5.4-mini",
		Messages: []chatMessage{
			{Role: "user", Content: "hello"},
		},
	}
	route := &routeResponse{
		PrimaryProvider:  "mock",
		PrimaryURL:       primaryURL,
		FallbackProvider: "mock",
		FallbackURL:      fallbackURL,
		FallbackModel:    "mock-fast",
		RetryCount:       0,
		TimeoutMS:        1000,
	}

	_, outTokens, retries, fallbackUsed, effectiveModel, errCode, err := s.executeWithResilience(context.Background(), req, route)
	if err != nil {
		t.Fatalf("executeWithResilience failed: err=%v errCode=%s", err, errCode)
	}
	if !fallbackUsed {
		t.Fatalf("expected fallbackUsed=true")
	}
	if retries != 0 {
		t.Fatalf("expected retries=0, got %d", retries)
	}
	if outTokens != 7 {
		t.Fatalf("expected completion tokens from fallback response, got %d", outTokens)
	}
	if effectiveModel != "mock-fast" {
		t.Fatalf("expected effective model mock-fast, got %q", effectiveModel)
	}
	if seenModel != "mock-fast" {
		t.Fatalf("expected fallback model mock-fast, got %q", seenModel)
	}
	if primaryCalls != 1 || fallbackCalls != 1 {
		t.Fatalf("unexpected call counts primary=%d fallback=%d", primaryCalls, fallbackCalls)
	}
}

func TestResolveRouteIncludesProjectID(t *testing.T) {
	var capturedQuery url.Values
	catalogURL := "http://catalog.test"

	s := &service{
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				capturedQuery = r.URL.Query()
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"model":"gpt-5.4-mini",
						"primary_provider":"openai",
						"fallback_provider":"mock",
						"fallback_model":"mock-fast",
						"primary_url":"http://primary",
						"fallback_url":"http://fallback",
						"timeout_ms":8000,
						"retry_count":1
					}`)),
					Header: make(http.Header),
				}, nil
			}),
		},
		providerCatalogURL: catalogURL,
	}

	route, err := s.resolveRoute(context.Background(), "gpt-5.4-mini", "project-123")
	if err != nil {
		t.Fatalf("resolveRoute failed: %v", err)
	}
	if route.FallbackModel != "mock-fast" {
		t.Fatalf("expected fallback_model=mock-fast, got %q", route.FallbackModel)
	}
	if got := capturedQuery.Get("project_id"); got != "project-123" {
		t.Fatalf("expected project_id in query, got %q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
