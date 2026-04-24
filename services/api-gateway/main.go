package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"llm-gateway-mvp/internal/platform/app"
	"llm-gateway-mvp/internal/platform/auth"
	"llm-gateway-mvp/internal/platform/db"
	"llm-gateway-mvp/internal/platform/events"
	"llm-gateway-mvp/internal/platform/httpx"
	lmt "llm-gateway-mvp/internal/platform/limits"
	"llm-gateway-mvp/internal/platform/util"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct {
	db                 *pgxpool.Pool
	projectKeyURL      string
	limitsURL          string
	providerCatalogURL string
	auditURL           string
	openAIKey          string
	publisher          *events.Publisher
	httpClient         *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type keyResolveResponse struct {
	KeyID     string `json:"key_id"`
	ProjectID string `json:"project_id"`
	OrgID     string `json:"org_id"`
	Status    string `json:"status"`
}

type routeResponse struct {
	Model            string `json:"model"`
	PrimaryProvider  string `json:"primary_provider"`
	FallbackProvider string `json:"fallback_provider"`
	FallbackModel    string `json:"fallback_model"`
	PrimaryURL       string `json:"primary_url"`
	FallbackURL      string `json:"fallback_url"`
	TimeoutMS        int    `json:"timeout_ms"`
	RetryCount       int    `json:"retry_count"`
}

type limitCheckRequest struct {
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id"`
	KeyID     string `json:"key_id"`
	Tokens    int64  `json:"tokens"`
}

type limitCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

func main() {
	ctx := context.Background()
	addr := env("HTTP_ADDR", ":8080")
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_gateway?sslmode=disable")

	pg, err := db.NewPGPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pg.Close()
	if err = ensureSchema(ctx, pg); err != nil {
		log.Fatalf("schema migrate: %v", err)
	}

	s := &service{
		db:                 pg,
		projectKeyURL:      env("PROJECT_KEY_SERVICE_URL", "http://localhost:8082"),
		limitsURL:          env("LIMITS_SERVICE_URL", "http://localhost:8083"),
		providerCatalogURL: env("PROVIDER_CATALOG_URL", "http://localhost:8084"),
		auditURL:           env("AUDIT_SERVICE_URL", "http://localhost:8085"),
		openAIKey:          env("OPENAI_API_KEY", ""),
		publisher:          events.NewPublisher(env("KAFKA_BROKERS", "")),
		httpClient:         &http.Client{Timeout: 20 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /v1/chat/completions", s.chatCompletions)

	app.RunHTTP(addr, mux)
}

func (s *service) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *service) chatCompletions(w http.ResponseWriter, r *http.Request) {
	requestID, _ := util.RandomToken(8)
	apiKey := auth.ExtractBearerToken(r)
	if apiKey == "" {
		httpx.Error(w, http.StatusUnauthorized, "missing_api_key", "Bearer API key is required")
		return
	}

	resolved, err := s.resolveKey(r.Context(), apiKey)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid_api_key", "API key is invalid")
		return
	}
	if resolved.Status != "active" {
		httpx.Error(w, http.StatusUnauthorized, "revoked_api_key", "API key is revoked")
		return
	}

	var req chatRequest
	if err = httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Model == "" {
		req.Model = "gpt-5.4-mini"
	}
	inputTokens := estimatePromptTokens(req)

	_ = s.publisher.Publish(r.Context(), "request.accepted", resolved.ProjectID, map[string]any{
		"request_id": requestID,
		"org_id":     resolved.OrgID,
		"project_id": resolved.ProjectID,
		"api_key_id": resolved.KeyID,
		"model":      req.Model,
	})

	allowed, reason, err := s.checkLimits(r.Context(), limitCheckRequest{
		OrgID:     resolved.OrgID,
		ProjectID: resolved.ProjectID,
		KeyID:     resolved.KeyID,
		Tokens:    int64(inputTokens),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "limit service unavailable")
		return
	}
	if !allowed {
		_ = s.publisher.Publish(r.Context(), "request.rejected", resolved.ProjectID, map[string]any{
			"request_id": requestID,
			"org_id":     resolved.OrgID,
			"project_id": resolved.ProjectID,
			"api_key_id": resolved.KeyID,
			"reason":     reason,
		})
		s.logTechnical(r.Context(), map[string]any{
			"request_id":     requestID,
			"org_id":         resolved.OrgID,
			"project_id":     resolved.ProjectID,
			"api_key_id":     resolved.KeyID,
			"model":          req.Model,
			"status":         "rejected",
			"error_code":     reason,
			"retries":        0,
			"fallback_used":  false,
			"fallback_model": "",
			"input_tokens":   inputTokens,
			"output_tokens":  0,
		})
		httpx.Error(w, http.StatusTooManyRequests, "limit_exceeded", reason)
		return
	}

	route, err := s.resolveRoute(r.Context(), req.Model, resolved.ProjectID)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "route_not_found", "route for model is not available")
		return
	}

	respBody, outputTokens, retries, fallbackUsed, effectiveModel, errCode, err := s.executeWithResilience(r.Context(), req, route)
	if err != nil {
		_ = s.publisher.Publish(r.Context(), "request.rejected", resolved.ProjectID, map[string]any{
			"request_id": requestID,
			"org_id":     resolved.OrgID,
			"project_id": resolved.ProjectID,
			"api_key_id": resolved.KeyID,
			"reason":     errCode,
		})
		s.logTechnical(r.Context(), map[string]any{
			"request_id":    requestID,
			"org_id":        resolved.OrgID,
			"project_id":    resolved.ProjectID,
			"api_key_id":    resolved.KeyID,
			"model":         req.Model,
			"status":        "failed",
			"error_code":    errCode,
			"retries":       retries,
			"fallback_used": fallbackUsed,
			"fallback_model": func() string {
				if fallbackUsed {
					return route.FallbackModel
				}
				return ""
			}(),
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		})
		httpx.Error(w, http.StatusBadGateway, "route_unavailable", "no available route for completion")
		return
	}
	if strings.TrimSpace(effectiveModel) == "" {
		effectiveModel = req.Model
	}

	totalTokens := inputTokens + outputTokens
	cost := s.estimateCost(r.Context(), effectiveModel, inputTokens, outputTokens)
	_ = s.publisher.Publish(r.Context(), "request.completed", resolved.ProjectID, map[string]any{
		"request_id":      requestID,
		"org_id":          resolved.OrgID,
		"project_id":      resolved.ProjectID,
		"api_key_id":      resolved.KeyID,
		"model":           req.Model,
		"effective_model": effectiveModel,
		"input_tokens":    inputTokens,
		"output_tokens":   outputTokens,
		"total_tokens":    totalTokens,
	})
	if fallbackUsed {
		_ = s.publisher.Publish(r.Context(), "fallback.used", resolved.ProjectID, map[string]any{
			"request_id":      requestID,
			"project_id":      resolved.ProjectID,
			"model":           req.Model,
			"effective_model": effectiveModel,
			"fallback_model":  route.FallbackModel,
		})
	}

	s.logTechnical(r.Context(), map[string]any{
		"request_id":    requestID,
		"org_id":        resolved.OrgID,
		"project_id":    resolved.ProjectID,
		"api_key_id":    resolved.KeyID,
		"model":         req.Model,
		"status":        "completed",
		"error_code":    "",
		"retries":       retries,
		"fallback_used": fallbackUsed,
		"fallback_model": func() string {
			if fallbackUsed {
				return route.FallbackModel
			}
			return ""
		}(),
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
	})

	s.recordUsage(r.Context(), map[string]any{
		"org_id":          resolved.OrgID,
		"project_id":      resolved.ProjectID,
		"api_key_id":      resolved.KeyID,
		"requested_model": req.Model,
		"effective_model": effectiveModel,
		"model":           effectiveModel,
		"input_tokens":    inputTokens,
		"output_tokens":   outputTokens,
		"total_tokens":    totalTokens,
		"estimated_cost":  cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func (s *service) resolveKey(ctx context.Context, apiKey string) (*keyResolveResponse, error) {
	endpoint := s.projectKeyURL + "/internal/keys/resolve?api_key=" + url.QueryEscape(apiKey)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("key not found")
	}
	var out keyResolveResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) checkLimits(ctx context.Context, reqBody limitCheckRequest) (bool, string, error) {
	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.limitsURL+"/internal/limits/check", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	var out limitCheckResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "", err
	}
	if resp.StatusCode == http.StatusOK {
		return true, "", nil
	}
	return false, out.Reason, nil
}

func (s *service) resolveRoute(ctx context.Context, model, projectID string) (*routeResponse, error) {
	query := s.providerCatalogURL + "/internal/route?model=" + url.QueryEscape(model)
	if strings.TrimSpace(projectID) != "" {
		query += "&project_id=" + url.QueryEscape(projectID)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("route not found")
	}
	var out routeResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.TimeoutMS < 500 {
		out.TimeoutMS = 500
	}
	if out.RetryCount < 0 {
		out.RetryCount = 0
	}
	return &out, nil
}

func (s *service) executeWithResilience(ctx context.Context, req chatRequest, route *routeResponse) ([]byte, int, int, bool, string, string, error) {
	payload, _ := json.Marshal(req)
	attempts := route.RetryCount + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		body, outTokens, err := s.callProvider(ctx, route.PrimaryProvider, route.PrimaryURL, payload, route.TimeoutMS)
		if err == nil {
			return body, outTokens, i, false, req.Model, "", nil
		}
		lastErr = err
	}

	if route.FallbackProvider != "" {
		fallbackReq := req
		if strings.TrimSpace(route.FallbackModel) != "" {
			fallbackReq.Model = strings.TrimSpace(route.FallbackModel)
		}
		fallbackPayload, _ := json.Marshal(fallbackReq)
		body, outTokens, err := s.callProvider(ctx, route.FallbackProvider, route.FallbackURL, fallbackPayload, route.TimeoutMS)
		if err == nil {
			return body, outTokens, attempts - 1, true, fallbackReq.Model, "", nil
		}
		lastErr = err
	}

	return nil, 0, attempts - 1, route.FallbackProvider != "", "", "route_unavailable", lastErr
}

func (s *service) callProvider(ctx context.Context, provider, url string, payload []byte, timeoutMS int) ([]byte, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if provider == "openai" {
		if s.openAIKey == "" {
			return nil, 0, errors.New("openai key not configured")
		}
		req.Header.Set("Authorization", "Bearer "+s.openAIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, errors.New("provider returned non-2xx")
	}
	return body, parseOutputTokens(body), nil
}

func parseOutputTokens(body []byte) int {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0
	}
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		return 0
	}
	if v, ok := usage["completion_tokens"]; ok {
		return toInt(v)
	}
	if v, ok := usage["total_tokens"]; ok {
		return toInt(v)
	}
	return 0
}

func estimatePromptTokens(req chatRequest) int {
	total := 0
	for _, m := range req.Messages {
		total += lmt.EstimateTokens(m.Content)
	}
	if total < 1 {
		total = 1
	}
	return total
}

func (s *service) estimateCost(ctx context.Context, model string, inputTokens, outputTokens int) float64 {
	var inputCost, outputCost float64
	err := s.db.QueryRow(ctx, `
		SELECT input_cost, output_cost
		FROM provider_models
		WHERE id = $1
	`, model).Scan(&inputCost, &outputCost)
	return estimatedCostByLookup(model, inputCost, outputCost, inputTokens, outputTokens, err)
}

func estimatedCostByLookup(model string, inputCost, outputCost float64, inputTokens, outputTokens int, err error) float64 {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("pricing not found for model=%s, using estimated_cost=0", model)
			return 0
		}
		log.Printf("cannot load pricing for model=%s: %v (using estimated_cost=0)", model, err)
		return 0
	}

	return estimatedCostFromRates(inputCost, outputCost, inputTokens, outputTokens)
}

func estimatedCostFromRates(inputCost, outputCost float64, inputTokens, outputTokens int) float64 {
	return float64(inputTokens)*inputCost/1000 + float64(outputTokens)*outputCost/1000
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

func (s *service) logTechnical(ctx context.Context, data map[string]any) {
	_, _ = s.db.Exec(ctx, `
		INSERT INTO technical_logs (
			request_id, org_id, project_id, api_key_id, model, status,
			error_code, retries, fallback_used, fallback_model, input_tokens, output_tokens
		) VALUES (
			$1, $2::uuid, $3::uuid, $4::uuid, $5, $6,
			$7, $8, $9, $10, $11, $12
		)
	`,
		data["request_id"],
		data["org_id"],
		data["project_id"],
		data["api_key_id"],
		data["model"],
		data["status"],
		data["error_code"],
		data["retries"],
		data["fallback_used"],
		data["fallback_model"],
		data["input_tokens"],
		data["output_tokens"],
	)
}

func (s *service) recordUsage(ctx context.Context, payload map[string]any) {
	_ = s.publisher.Publish(ctx, "usage.recorded", payload["project_id"].(string), payload)
	if s.auditURL == "" {
		return
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.auditURL+"/internal/usage/record", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, _ = s.httpClient.Do(req)
}

func env(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func ensureSchema(ctx context.Context, pg *pgxpool.Pool) error {
	_, err := pg.Exec(ctx, `ALTER TABLE technical_logs ADD COLUMN IF NOT EXISTS fallback_model TEXT`)
	return err
}
