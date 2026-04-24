package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"llm-gateway-mvp/internal/platform/app"
	"llm-gateway-mvp/internal/platform/auth"
	"llm-gateway-mvp/internal/platform/db"
	"llm-gateway-mvp/internal/platform/httpx"
	lmt "llm-gateway-mvp/internal/platform/limits"
	"llm-gateway-mvp/internal/platform/util"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct {
	db         *pgxpool.Pool
	openAIURL  string
	openAIKey  string
	geminiURL  string
	geminiKey  string
	selfURL    string
	httpClient *http.Client

	statusTTL   time.Duration
	statusMu    sync.RWMutex
	statusCache map[string]providerStatus
}

type providerStatus struct {
	Status    string
	CheckedAt time.Time
	LatencyMS int64
	Error     string
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type modelPricingUpdateRequest struct {
	InputCost     *float64 `json:"input_cost"`
	OutputCost    *float64 `json:"output_cost"`
	PricingSource string   `json:"pricing_source"`
}

func main() {
	ctx := context.Background()
	addr := env("HTTP_ADDR", ":8084")
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_gateway?sslmode=disable")
	secret := env("JWT_SECRET", "dev-secret")
	openAIURL := env("OPENAI_API_URL", "https://api.openai.com/v1/chat/completions")
	openAIKey := strings.TrimSpace(env("OPENAI_API_KEY", ""))
	geminiURL := env("GEMINI_API_URL", "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent")
	geminiKey := strings.TrimSpace(env("GEMINI_API_KEY", ""))
	selfURL := env("PROVIDER_CATALOG_URL", "http://provider-catalog-service:8084")

	pg, err := db.NewPGPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pg.Close()
	if err = ensureSchema(ctx, pg); err != nil {
		log.Fatalf("schema migrate: %v", err)
	}

	s := &service{
		db:          pg,
		openAIURL:   openAIURL,
		openAIKey:   openAIKey,
		geminiURL:   geminiURL,
		geminiKey:   geminiKey,
		selfURL:     selfURL,
		httpClient:  &http.Client{Timeout: 20 * time.Second},
		statusTTL:   30 * time.Second,
		statusCache: make(map[string]providerStatus),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /catalog/models", s.models)
	mux.HandleFunc("GET /catalog/providers/status", s.providersStatus)
	mux.Handle("PUT /catalog/models/{id}/pricing", auth.Middleware(secret, "Admin")(http.HandlerFunc(s.updateModelPricing)))
	mux.HandleFunc("GET /internal/route", s.route)
	mux.HandleFunc("POST /internal/mock/completions", s.mockCompletions)
	mux.HandleFunc("POST /internal/gemini/completions", s.geminiCompletions)

	app.RunHTTP(addr, mux)
}

func (s *service) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *service) models(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		SELECT id, provider, status, input_cost, output_cost, pricing_source, pricing_updated_at
		FROM provider_models
		ORDER BY provider, id
	`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query models")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, provider, status, pricingSource string
		var inputCost, outputCost float64
		var pricingUpdatedAt time.Time
		if err = rows.Scan(&id, &provider, &status, &inputCost, &outputCost, &pricingSource, &pricingUpdatedAt); err == nil {
			items = append(items, map[string]any{
				"id":                 id,
				"provider":           provider,
				"status":             status,
				"input_cost":         inputCost,
				"output_cost":        outputCost,
				"pricing_source":     pricingSource,
				"pricing_updated_at": pricingUpdatedAt.UTC(),
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) providersStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		SELECT DISTINCT provider
		FROM provider_models
		ORDER BY provider
	`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query provider list")
		return
	}
	defer rows.Close()

	forceRefresh := parseRefreshQuery(r.URL.Query().Get("refresh"))
	items := make([]map[string]any, 0)
	for rows.Next() {
		var provider string
		if err = rows.Scan(&provider); err != nil {
			continue
		}
		status := s.getProviderStatus(r.Context(), provider, forceRefresh)
		items = append(items, map[string]any{
			"provider":   provider,
			"status":     status.Status,
			"checked_at": status.CheckedAt.UTC(),
			"latency_ms": status.LatencyMS,
			"error":      status.Error,
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) getProviderStatus(ctx context.Context, provider string, forceRefresh bool) providerStatus {
	now := time.Now().UTC()
	s.statusMu.RLock()
	cached, ok := s.statusCache[provider]
	s.statusMu.RUnlock()

	if ok && !forceRefresh && now.Sub(cached.CheckedAt) < s.statusTTL {
		return cached
	}

	fresh := s.probeProvider(ctx, provider)
	if fresh.CheckedAt.IsZero() {
		fresh.CheckedAt = now
	}

	s.statusMu.Lock()
	s.statusCache[provider] = fresh
	s.statusMu.Unlock()

	return fresh
}

func (s *service) probeProvider(ctx context.Context, provider string) providerStatus {
	start := time.Now().UTC()
	result := providerStatus{
		Status:    "down",
		CheckedAt: start,
		LatencyMS: 0,
		Error:     "",
	}

	var err error
	switch provider {
	case "mock":
		result.Status = "up"
		return result
	case "openai":
		err = s.probeOpenAI(ctx)
	case "gemini":
		err = s.probeGemini(ctx)
	default:
		err = errors.New("unknown provider")
	}

	result.LatencyMS = time.Since(start).Milliseconds()
	if err == nil {
		result.Status = "up"
		return result
	}
	result.Status = "down"
	result.Error = err.Error()
	return result
}

func (s *service) probeOpenAI(ctx context.Context) error {
	if strings.TrimSpace(s.openAIKey) == "" {
		return errors.New("OPENAI_API_KEY is not configured")
	}
	probeURL, err := openAIProbeURL(s.openAIURL)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, probeURL, nil)
	req.Header.Set("Authorization", "Bearer "+s.openAIKey)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return errors.New("openai probe failed: " + providerError(resp))
}

func (s *service) probeGemini(ctx context.Context) error {
	if strings.TrimSpace(s.geminiKey) == "" {
		return errors.New("GEMINI_API_KEY is not configured")
	}
	probeURL, err := geminiProbeURL(s.geminiURL, s.geminiKey)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, probeURL, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return errors.New("gemini probe failed: " + providerError(resp))
}

func openAIProbeURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("invalid OPENAI_API_URL")
	}
	u.Path = "/v1/models"
	u.RawQuery = ""
	return u.String(), nil
}

func geminiProbeURL(raw, apiKey string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("invalid GEMINI_API_URL")
	}
	u.Path = "/v1beta/models"
	q := u.Query()
	if q.Get("key") == "" {
		q.Set("key", apiKey)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func providerError(resp *http.Response) string {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "http " + strconv.Itoa(resp.StatusCode)
	}
	return "http " + strconv.Itoa(resp.StatusCode) + ": " + trimmed
}

func parseRefreshQuery(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func blendedUSDPerToken(inputCost, outputCost float64) float64 {
	return (inputCost + outputCost) / 2 / 1000
}

func formatUSDPerToken(value float64) string {
	rounded := math.Round(value*1e12) / 1e12
	return strconv.FormatFloat(rounded, 'f', 12, 64)
}

func (s *service) updateModelPricing(w http.ResponseWriter, r *http.Request) {
	modelID := strings.TrimSpace(r.PathValue("id"))
	if modelID == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "model id is required")
		return
	}

	var req modelPricingUpdateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.InputCost == nil || req.OutputCost == nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "input_cost and output_cost are required")
		return
	}
	if math.IsNaN(*req.InputCost) || math.IsInf(*req.InputCost, 0) || *req.InputCost < 0 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "input_cost must be a non-negative number")
		return
	}
	if math.IsNaN(*req.OutputCost) || math.IsInf(*req.OutputCost, 0) || *req.OutputCost < 0 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "output_cost must be a non-negative number")
		return
	}

	source := strings.TrimSpace(req.PricingSource)
	if source == "" {
		source = "manual"
	}
	if len(source) > 64 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "pricing_source is too long")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot begin transaction")
		return
	}
	defer tx.Rollback(r.Context())

	var provider, status, pricingSource string
	var inputCost, outputCost float64
	var pricingUpdatedAt time.Time
	err = tx.QueryRow(r.Context(), `
		UPDATE provider_models
		SET input_cost = $2,
		    output_cost = $3,
		    pricing_source = $4,
		    pricing_updated_at = NOW()
		WHERE id = $1
		RETURNING provider, status, input_cost, output_cost, pricing_source, pricing_updated_at
	`, modelID, *req.InputCost, *req.OutputCost, source).Scan(&provider, &status, &inputCost, &outputCost, &pricingSource, &pricingUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "not_found", "model not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot update pricing")
		return
	}

	usdPerTokenValue := blendedUSDPerToken(inputCost, outputCost)
	usdPerToken := formatUSDPerToken(usdPerTokenValue)
	affectedProjectLimits, err := s.autoSyncProjectLimits(r.Context(), tx, modelID, usdPerToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot autosync project limits")
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot commit pricing update")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":                      modelID,
		"provider":                provider,
		"status":                  status,
		"input_cost":              inputCost,
		"output_cost":             outputCost,
		"pricing_source":          pricingSource,
		"pricing_updated_at":      pricingUpdatedAt.UTC(),
		"affected_project_limits": affectedProjectLimits,
	})
}

func (s *service) autoSyncProjectLimits(ctx context.Context, tx pgx.Tx, modelID string, usdPerToken string) (int64, error) {
	res, err := tx.Exec(ctx, `
		UPDATE limits
		SET
			token_limit = CASE
				WHEN COALESCE(NULLIF(sync_source, ''), 'tokens') = 'budget'
				     AND $2::numeric > 0
				     AND COALESCE(budget_limit_usd, 0) > 0
				THEN FLOOR(COALESCE(budget_limit_usd, 0)::numeric / $2::numeric)::bigint
				ELSE token_limit
			END,
			budget_limit_usd = CASE
				WHEN COALESCE(NULLIF(sync_source, ''), 'tokens') = 'tokens'
				THEN ROUND((token_limit::numeric * $2::numeric), 12)
				ELSE budget_limit_usd
			END,
			usd_per_token = $2::numeric,
			updated_at = NOW()
		WHERE scope_type = 'project'
		  AND billing_model = $1
	`, modelID, usdPerToken)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

func (s *service) route(w http.ResponseWriter, r *http.Request) {
	model := strings.TrimSpace(r.URL.Query().Get("model"))
	if model == "" {
		model = "gpt-5.4-mini"
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))

	var primaryProvider, fallbackProvider string
	var timeoutMS, retryCount int
	err := s.db.QueryRow(r.Context(), `
		SELECT primary_provider, fallback_provider, timeout_ms, retry_count
		FROM routing_rules
		WHERE model_id = $1
	`, model).Scan(&primaryProvider, &fallbackProvider, &timeoutMS, &retryCount)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "route not found")
		return
	}

	fallbackModel := ""
	if projectID != "" {
		var overrideFallbackModelID *string
		err = s.db.QueryRow(r.Context(), `
			SELECT fallback_model_id
			FROM project_routing_settings
			WHERE project_id = $1::uuid
		`, projectID).Scan(&overrideFallbackModelID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot resolve project routing")
			return
		}
		if err == nil && overrideFallbackModelID != nil && strings.TrimSpace(*overrideFallbackModelID) != "" {
			fallbackModel = strings.TrimSpace(*overrideFallbackModelID)
			var overrideFallbackProvider string
			err = s.db.QueryRow(r.Context(), `
				SELECT primary_provider
				FROM routing_rules
				WHERE model_id = $1
			`, fallbackModel).Scan(&overrideFallbackProvider)
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, "validation_error", "fallback model route is not configured")
				return
			}
			fallbackProvider = overrideFallbackProvider
		}
	}

	primaryURL := s.providerURL(primaryProvider)
	if primaryURL == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "primary provider route is not configured")
		return
	}
	fallbackURL := ""
	if fallbackProvider != "" {
		fallbackURL = s.providerURL(fallbackProvider)
		if fallbackURL == "" {
			httpx.Error(w, http.StatusBadRequest, "validation_error", "fallback provider route is not configured")
			return
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"model":             model,
		"primary_provider":  primaryProvider,
		"fallback_provider": fallbackProvider,
		"fallback_model":    fallbackModel,
		"timeout_ms":        timeoutMS,
		"retry_count":       retryCount,
		"primary_url":       primaryURL,
		"fallback_url":      fallbackURL,
	})
}

func (s *service) mockCompletions(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Model == "" {
		req.Model = "mock-fast"
	}
	prompt := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			prompt = req.Messages[i].Content
			break
		}
	}
	if prompt == "" {
		prompt = "No user prompt provided"
	}

	answer := "[mock-fallback] " + prompt
	inputTokens := lmt.EstimateTokens(prompt)
	outputTokens := lmt.EstimateTokens(answer)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":      "mock-" + time.Now().UTC().Format("20060102150405") + "-" + util.SHA256(prompt)[:8],
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": answer,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     inputTokens,
			"completion_tokens": outputTokens,
			"total_tokens":      inputTokens + outputTokens,
		},
	})
}

func (s *service) geminiCompletions(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Model == "" {
		req.Model = "gemini-2.5-flash"
	}
	if strings.TrimSpace(s.geminiKey) == "" {
		httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "gemini api key is not configured")
		return
	}

	body, _ := json.Marshal(map[string]any{
		"contents": toGeminiContents(req.Messages),
	})
	upstreamURL, err := url.Parse(s.geminiURL)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "invalid gemini api url")
		return
	}
	query := upstreamURL.Query()
	if query.Get("key") == "" {
		query.Set("key", s.geminiKey)
	}
	upstreamURL.RawQuery = query.Encode()

	upReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL.String(), bytes.NewReader(body))
	upReq.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(upReq)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "gemini request failed")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "cannot read gemini response")
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "gemini returned non-2xx")
		return
	}

	type geminiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	var payload geminiResponse
	if err = json.Unmarshal(respBody, &payload); err != nil {
		httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "invalid gemini response")
		return
	}

	answer := ""
	if len(payload.Candidates) > 0 && len(payload.Candidates[0].Content.Parts) > 0 {
		answer = strings.TrimSpace(payload.Candidates[0].Content.Parts[0].Text)
	}
	if answer == "" {
		answer = "[gemini] empty content"
	}

	promptTokens := payload.UsageMetadata.PromptTokenCount
	completionTokens := payload.UsageMetadata.CandidatesTokenCount
	totalTokens := payload.UsageMetadata.TotalTokenCount
	if promptTokens <= 0 {
		promptTokens = estimatePromptTokens(req.Messages)
	}
	if completionTokens <= 0 {
		completionTokens = lmt.EstimateTokens(answer)
	}
	if totalTokens <= 0 {
		totalTokens = promptTokens + completionTokens
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":      "gemini-" + time.Now().UTC().Format("20060102150405") + "-" + util.SHA256(answer)[:8],
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": answer,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
		},
	})
}

func (s *service) providerURL(provider string) string {
	switch provider {
	case "openai":
		return s.openAIURL
	case "gemini":
		return strings.TrimRight(s.selfURL, "/") + "/internal/gemini/completions"
	case "mock":
		return strings.TrimRight(s.selfURL, "/") + "/internal/mock/completions"
	default:
		return ""
	}
}

func toGeminiContents(messages []chatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		text := strings.TrimSpace(msg.Content)
		if text == "" {
			continue
		}
		role := "user"
		switch msg.Role {
		case "assistant":
			role = "model"
		case "system":
			role = "user"
			text = "System instruction: " + text
		default:
			role = "user"
		}
		out = append(out, map[string]any{
			"role": role,
			"parts": []map[string]any{
				{"text": text},
			},
		})
	}
	if len(out) == 0 {
		out = append(out, map[string]any{
			"role": "user",
			"parts": []map[string]any{
				{"text": "No user prompt provided"},
			},
		})
	}
	return out
}

func estimatePromptTokens(messages []chatMessage) int {
	total := 0
	for _, msg := range messages {
		total += lmt.EstimateTokens(msg.Content)
	}
	if total < 1 {
		return 1
	}
	return total
}

func env(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func ensureSchema(ctx context.Context, pg *pgxpool.Pool) error {
	_, err := pg.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS project_routing_settings (
			project_id UUID PRIMARY KEY REFERENCES projects(id),
			fallback_model_id TEXT REFERENCES provider_models(id),
			updated_by UUID NOT NULL REFERENCES users(id),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		ALTER TABLE provider_models
		ADD COLUMN IF NOT EXISTS pricing_source TEXT NOT NULL DEFAULT 'seed';

		ALTER TABLE provider_models
		ADD COLUMN IF NOT EXISTS pricing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

		ALTER TABLE limits
		ADD COLUMN IF NOT EXISTS sync_source TEXT NOT NULL DEFAULT 'tokens';

		UPDATE provider_models
		SET pricing_source = 'seed'
		WHERE pricing_source IS NULL OR pricing_source = '';

		UPDATE provider_models
		SET pricing_updated_at = COALESCE(pricing_updated_at, created_at, NOW())
		WHERE pricing_updated_at IS NULL;

		UPDATE limits
		SET sync_source = 'tokens'
		WHERE sync_source IS NULL OR sync_source = '';
	`)
	return err
}
