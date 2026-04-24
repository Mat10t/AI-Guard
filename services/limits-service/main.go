package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type service struct {
	db         *pgxpool.Pool
	redis      *redis.Client
	secret     string
	auditURL   string
	publisher  *events.Publisher
	httpClient *http.Client
}

type setLimitRequest struct {
	TokenLimit int64  `json:"token_limit"`
	Period     string `json:"period"`
}

type setProjectLimitRequest struct {
	TokenLimit     *int64   `json:"token_limit,omitempty"`
	BudgetLimitUSD *float64 `json:"budget_limit_usd,omitempty"`
	BillingModel   string   `json:"billing_model,omitempty"`
	Period         string   `json:"period"`
	SyncSource     string   `json:"sync_source,omitempty"`
}

type checkLimitRequest struct {
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id"`
	KeyID     string `json:"key_id"`
	Tokens    int64  `json:"tokens"`
}

type projectLimitRecord struct {
	TokenLimit     int64
	Period         string
	BudgetLimitUSD float64
	BillingModel   string
	USDPerToken    float64
	SyncSource     string
}

func main() {
	ctx := context.Background()

	addr := env("HTTP_ADDR", ":8083")
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_gateway?sslmode=disable")
	redisAddr := env("REDIS_ADDR", "localhost:6379")
	secret := env("JWT_SECRET", "dev-secret")
	auditURL := env("AUDIT_SERVICE_URL", "http://localhost:8085")
	brokers := env("KAFKA_BROKERS", "")

	pg, err := db.NewPGPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pg.Close()
	if err = ensureSchema(ctx, pg); err != nil {
		log.Fatalf("schema migrate: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err = rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis not reachable on startup: %v", err)
	}

	s := &service{
		db:         pg,
		redis:      rdb,
		secret:     secret,
		auditURL:   auditURL,
		publisher:  events.NewPublisher(brokers),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.Handle("GET /limits/projects/{id}", auth.Middleware(secret)(http.HandlerFunc(s.getProjectLimit)))
	mux.Handle("PUT /limits/projects/{id}", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.setProjectLimit)))
	mux.Handle("PUT /limits/keys/{id}", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.setKeyLimit)))
	mux.HandleFunc("POST /internal/limits/check", s.check)

	app.RunHTTP(addr, mux)
}

func (s *service) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *service) setProjectLimit(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	if projectID == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "project id is required")
		return
	}
	if err := s.ensureProjectAccess(r.Context(), projectID, claims.OrgID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot check project access")
		return
	}

	var req setProjectLimitRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}

	existing, _ := s.loadProjectLimit(r.Context(), projectID, claims.OrgID)

	period := strings.TrimSpace(req.Period)
	if period == "" && existing != nil {
		period = existing.Period
	}
	if !lmt.ValidatePeriod(period) {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "period must be day/week/month")
		return
	}

	billingModel := strings.TrimSpace(req.BillingModel)
	if billingModel == "" && existing != nil {
		billingModel = strings.TrimSpace(existing.BillingModel)
	}
	if billingModel == "" {
		billingModel = "gpt-5.4-mini"
	}
	usdPerToken, err := s.getModelUSDPerToken(r.Context(), billingModel)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot load model pricing")
		return
	}

	syncSource := normalizeSyncSource(req.SyncSource)
	if strings.TrimSpace(req.SyncSource) == "" && existing != nil {
		syncSource = normalizeSyncSource(existing.SyncSource)
	}
	var tokenLimit int64
	var budgetLimitUSD float64

	switch syncSource {
	case "budget":
		var budget float64
		switch {
		case req.BudgetLimitUSD != nil:
			budget = *req.BudgetLimitUSD
		case existing != nil:
			budget = existing.BudgetLimitUSD
		}
		if budget <= 0 {
			httpx.Error(w, http.StatusBadRequest, "validation_error", "budget_limit_usd must be > 0")
			return
		}
		if usdPerToken <= 0 {
			httpx.Error(w, http.StatusBadRequest, "validation_error", "budget conversion is unavailable for selected model")
			return
		}
		tokenLimit = computeTokensFromBudget(budget, usdPerToken)
		if tokenLimit <= 0 {
			httpx.Error(w, http.StatusBadRequest, "validation_error", "budget is too small for selected model")
			return
		}
		budgetLimitUSD = roundBudgetUSD(budget)
	default:
		switch {
		case req.TokenLimit != nil:
			tokenLimit = *req.TokenLimit
		case existing != nil:
			tokenLimit = existing.TokenLimit
		}
		if tokenLimit <= 0 {
			httpx.Error(w, http.StatusBadRequest, "validation_error", "token_limit must be > 0")
			return
		}
		budgetLimitUSD = computeBudgetFromTokens(tokenLimit, usdPerToken)
	}

	_, err = s.db.Exec(r.Context(), `
		INSERT INTO limits (
			id, scope_type, scope_id, token_limit, period, budget_limit_usd, billing_model, usd_per_token, sync_source, created_by, updated_at
		)
		VALUES (gen_random_uuid(), 'project', $1::uuid, $2, $3, $4, $5, $6, $7, $8::uuid, NOW())
		ON CONFLICT (scope_type, scope_id)
		DO UPDATE SET
			token_limit = EXCLUDED.token_limit,
			period = EXCLUDED.period,
			budget_limit_usd = EXCLUDED.budget_limit_usd,
			billing_model = EXCLUDED.billing_model,
			usd_per_token = EXCLUDED.usd_per_token,
			sync_source = EXCLUDED.sync_source,
			created_by = EXCLUDED.created_by,
			updated_at = NOW()
	`, projectID, tokenLimit, period, budgetLimitUSD, billingModel, usdPerToken, syncSource, claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot save project limit")
		return
	}

	evt := map[string]any{
		"scope_type":       "project",
		"scope_id":         projectID,
		"token_limit":      tokenLimit,
		"budget_limit_usd": budgetLimitUSD,
		"billing_model":    billingModel,
		"usd_per_token":    usdPerToken,
		"period":           period,
		"sync_source":      syncSource,
		"org_id":           claims.OrgID,
		"actor":            claims.UserID,
	}
	_ = s.publisher.Publish(r.Context(), "limit.updated", projectID, evt)
	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"actor_user_id": claims.UserID,
		"action":        "limit.updated",
		"object_type":   "project_limit",
		"object_id":     projectID,
		"details":       evt,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"scope_type":       "project",
		"scope_id":         projectID,
		"token_limit":      tokenLimit,
		"budget_limit_usd": budgetLimitUSD,
		"billing_model":    billingModel,
		"usd_per_token":    usdPerToken,
		"period":           period,
		"sync_source":      syncSource,
	})
}

func (s *service) setKeyLimit(w http.ResponseWriter, r *http.Request) {
	s.setLimit(w, r, "key", r.PathValue("id"))
}

func (s *service) getProjectLimit(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	if projectID == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "project id is required")
		return
	}
	limit, err := s.loadProjectLimit(r.Context(), projectID, claims.OrgID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "not_found", "project limit not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot load project limit")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"scope_type":       "project",
		"scope_id":         projectID,
		"token_limit":      limit.TokenLimit,
		"budget_limit_usd": limit.BudgetLimitUSD,
		"billing_model":    limit.BillingModel,
		"usd_per_token":    limit.USDPerToken,
		"period":           limit.Period,
		"sync_source":      limit.SyncSource,
	})
}

func (s *service) setLimit(w http.ResponseWriter, r *http.Request, scopeType, scopeID string) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	var req setLimitRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.TokenLimit <= 0 || !lmt.ValidatePeriod(req.Period) {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "token_limit must be >0 and period must be day/week/month")
		return
	}

	_, err := s.db.Exec(r.Context(), `
		INSERT INTO limits (id, scope_type, scope_id, token_limit, period, budget_limit_usd, billing_model, usd_per_token, created_by, updated_at)
		VALUES (gen_random_uuid(), $1, $2::uuid, $3, $4, NULL, NULL, NULL, $5::uuid, NOW())
		ON CONFLICT (scope_type, scope_id)
		DO UPDATE SET
			token_limit = EXCLUDED.token_limit,
			period = EXCLUDED.period,
			budget_limit_usd = NULL,
			billing_model = NULL,
			usd_per_token = NULL,
			created_by = EXCLUDED.created_by,
			updated_at = NOW()
	`, scopeType, scopeID, req.TokenLimit, req.Period, claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot save limit")
		return
	}

	evt := map[string]any{
		"scope_type":  scopeType,
		"scope_id":    scopeID,
		"token_limit": req.TokenLimit,
		"period":      req.Period,
		"org_id":      claims.OrgID,
		"actor":       claims.UserID,
	}
	projectIDForKey := ""
	if scopeType == "project" {
		evt["project_id"] = scopeID
	}
	if scopeType == "key" {
		evt["api_key_id"] = scopeID
		if resolvedProjectID, resolveErr := s.resolveProjectIDForKey(r.Context(), scopeID, claims.OrgID); resolveErr == nil {
			projectIDForKey = resolvedProjectID
			if projectIDForKey != "" {
				evt["project_id"] = projectIDForKey
			}
		}
	}
	_ = s.publisher.Publish(r.Context(), "limit.updated", scopeID, evt)
	auditPayload := map[string]any{
		"org_id":        claims.OrgID,
		"actor_user_id": claims.UserID,
		"action":        "limit.updated",
		"object_type":   scopeType + "_limit",
		"object_id":     scopeID,
		"details":       evt,
	}
	if scopeType == "project" {
		auditPayload["project_id"] = scopeID
	}
	if scopeType == "key" {
		auditPayload["api_key_id"] = scopeID
		if projectIDForKey != "" {
			auditPayload["project_id"] = projectIDForKey
		}
	}
	s.publishAudit(r.Context(), auditPayload)

	httpx.JSON(w, http.StatusOK, map[string]any{"scope_type": scopeType, "scope_id": scopeID, "token_limit": req.TokenLimit, "period": req.Period})
}

func (s *service) check(w http.ResponseWriter, r *http.Request) {
	var req checkLimitRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.ProjectID == "" || req.KeyID == "" || req.Tokens <= 0 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "project_id, key_id and tokens >0 are required")
		return
	}

	allowed, reason, err := s.checkScope(r.Context(), "project", req.ProjectID, req.Tokens)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot check project limit")
		return
	}
	if !allowed {
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"allowed": false, "reason": reason})
		return
	}

	allowed, reason, err = s.checkScope(r.Context(), "key", req.KeyID, req.Tokens)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot check key limit")
		return
	}
	if !allowed {
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"allowed": false, "reason": reason})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"allowed": true})
}

func (s *service) checkScope(ctx context.Context, scopeType, scopeID string, tokens int64) (bool, string, error) {
	var tokenLimit int64
	var period string
	err := s.db.QueryRow(ctx, `
		SELECT token_limit, period
		FROM limits
		WHERE scope_type = $1 AND scope_id = $2::uuid
	`, scopeType, scopeID).Scan(&tokenLimit, &period)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, "db_error", err
		}
		return true, "", nil
	}

	bucket, err := lmt.Bucket(time.Now(), period)
	if err != nil {
		return false, "invalid_period", err
	}
	key := fmt.Sprintf("limit:%s:%s:%s", scopeType, scopeID, bucket)
	currentStr, err := s.redis.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return false, "redis_error", err
	}
	var current int64
	if currentStr != "" {
		current, _ = strconv.ParseInt(currentStr, 10, 64)
	}
	if current+tokens > tokenLimit {
		return false, scopeType + "_limit_exceeded", nil
	}

	pipe := s.redis.TxPipeline()
	pipe.IncrBy(ctx, key, tokens)
	pipe.Expire(ctx, key, periodTTL(period))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, "redis_error", err
	}

	return true, "", nil
}

func (s *service) publishAudit(ctx context.Context, payload map[string]any) {
	if s.auditURL == "" {
		return
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.auditURL+"/internal/audit/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, _ = s.httpClient.Do(req)
}

func periodTTL(period string) time.Duration {
	switch period {
	case "day":
		return 24 * time.Hour
	case "week":
		return 7 * 24 * time.Hour
	case "month":
		return 31 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func env(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func ensureSchema(ctx context.Context, pg *pgxpool.Pool) error {
	queries := []string{
		`ALTER TABLE limits ADD COLUMN IF NOT EXISTS budget_limit_usd NUMERIC(18,12)`,
		`ALTER TABLE limits ADD COLUMN IF NOT EXISTS billing_model TEXT`,
		`ALTER TABLE limits ADD COLUMN IF NOT EXISTS usd_per_token NUMERIC(18,12)`,
		`ALTER TABLE limits ADD COLUMN IF NOT EXISTS sync_source TEXT NOT NULL DEFAULT 'tokens'`,
		`UPDATE limits SET sync_source = 'tokens' WHERE sync_source IS NULL OR sync_source = ''`,
	}
	for _, q := range queries {
		if _, err := pg.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func normalizeSyncSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "budget":
		return "budget"
	default:
		return "tokens"
	}
}

func blendedUSDPerToken(inputCost, outputCost float64) float64 {
	return (inputCost + outputCost) / 2 / 1000
}

func computeBudgetFromTokens(tokens int64, usdPerToken float64) float64 {
	if tokens <= 0 || usdPerToken <= 0 {
		return 0
	}
	return roundBudgetUSD(float64(tokens) * usdPerToken)
}

func computeTokensFromBudget(budgetUSD, usdPerToken float64) int64 {
	if budgetUSD <= 0 || usdPerToken <= 0 {
		return 0
	}
	return int64(math.Floor(budgetUSD / usdPerToken))
}

func roundBudgetUSD(value float64) float64 {
	return math.Round(value*1e12) / 1e12
}

func (s *service) getModelUSDPerToken(ctx context.Context, modelID string) (float64, error) {
	var inputCost, outputCost float64
	err := s.db.QueryRow(ctx, `
		SELECT input_cost, output_cost
		FROM provider_models
		WHERE id = $1
	`, modelID).Scan(&inputCost, &outputCost)
	if err != nil {
		return 0, err
	}
	return blendedUSDPerToken(inputCost, outputCost), nil
}

func (s *service) resolveProjectIDForKey(ctx context.Context, keyID, orgID string) (string, error) {
	var projectID string
	err := s.db.QueryRow(ctx, `
		SELECT p.id::text
		FROM api_keys k
		JOIN projects p ON p.id = k.project_id
		WHERE k.id = $1::uuid
		  AND p.org_id = $2::uuid
		  AND p.deleted_at IS NULL
	`, keyID, orgID).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return projectID, nil
}

func (s *service) loadProjectLimit(ctx context.Context, projectID, orgID string) (*projectLimitRecord, error) {
	var item projectLimitRecord
	err := s.db.QueryRow(ctx, `
		SELECT token_limit, period, COALESCE(budget_limit_usd, 0), COALESCE(billing_model, ''), COALESCE(usd_per_token, 0), COALESCE(sync_source, 'tokens')
		FROM limits l
		JOIN projects p ON p.id = l.scope_id
		WHERE l.scope_type = 'project'
		  AND l.scope_id = $1::uuid
		  AND p.org_id = $2::uuid
		  AND p.deleted_at IS NULL
	`, projectID, orgID).Scan(
		&item.TokenLimit,
		&item.Period,
		&item.BudgetLimitUSD,
		&item.BillingModel,
		&item.USDPerToken,
		&item.SyncSource,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *service) ensureProjectAccess(ctx context.Context, projectID, orgID string) error {
	var exists int
	return s.db.QueryRow(ctx, `
		SELECT 1
		FROM projects
		WHERE id = $1::uuid AND org_id = $2::uuid AND deleted_at IS NULL
	`, projectID, orgID).Scan(&exists)
}
