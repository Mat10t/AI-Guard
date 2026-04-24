package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"llm-gateway-mvp/internal/platform/app"
	"llm-gateway-mvp/internal/platform/auth"
	"llm-gateway-mvp/internal/platform/db"
	"llm-gateway-mvp/internal/platform/httpx"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct {
	db     *pgxpool.Pool
	secret string
}

type auditEvent struct {
	OrgID       string         `json:"org_id"`
	ProjectID   string         `json:"project_id"`
	APIKeyID    string         `json:"api_key_id"`
	ActorUserID string         `json:"actor_user_id"`
	Action      string         `json:"action"`
	ObjectType  string         `json:"object_type"`
	ObjectID    string         `json:"object_id"`
	Details     map[string]any `json:"details"`
}

type usageEvent struct {
	OrgID          string  `json:"org_id"`
	ProjectID      string  `json:"project_id"`
	APIKeyID       string  `json:"api_key_id"`
	Model          string  `json:"model"`
	RequestedModel string  `json:"requested_model"`
	EffectiveModel string  `json:"effective_model"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	TotalTokens    int     `json:"total_tokens"`
	EstimatedCost  float64 `json:"estimated_cost"`
}

type scopeSelection struct {
	Scope     string
	ProjectID string
	APIKeyID  string
}

func main() {
	addr := env("HTTP_ADDR", ":8085")
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_gateway?sslmode=disable")
	secret := env("JWT_SECRET", "dev-secret")

	pg, err := db.NewPGPool(rctx(), dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pg.Close()
	if err = ensureSchema(rctx(), pg); err != nil {
		log.Fatalf("schema migrate: %v", err)
	}

	s := &service{db: pg, secret: secret}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /internal/audit/event", s.ingestAudit)
	mux.HandleFunc("POST /internal/usage/record", s.ingestUsage)
	mux.Handle("GET /audit", auth.Middleware(secret, "Admin", "PM", "Finance")(http.HandlerFunc(s.auditList)))
	mux.Handle("GET /logs/technical", auth.Middleware(secret, "Admin", "PM", "Dev", "Finance")(http.HandlerFunc(s.technicalLogs)))
	mux.Handle("GET /analytics/usage", auth.Middleware(secret, "Admin", "PM", "Dev", "Finance")(http.HandlerFunc(s.analytics)))
	mux.Handle("GET /analytics/timeseries", auth.Middleware(secret, "Admin", "PM", "Dev", "Finance")(http.HandlerFunc(s.analyticsTimeseries)))
	mux.Handle("GET /reports/csv/usage", auth.Middleware(secret, "Admin", "PM", "Finance")(http.HandlerFunc(s.exportCSVUsage)))
	mux.Handle("GET /reports/csv/audit", auth.Middleware(secret, "Admin", "PM", "Finance")(http.HandlerFunc(s.exportCSVAudit)))
	mux.Handle("GET /reports/csv/logs", auth.Middleware(secret, "Admin", "PM", "Finance")(http.HandlerFunc(s.exportCSVLogs)))
	mux.Handle("GET /reports/csv", auth.Middleware(secret, "Admin", "PM", "Finance")(http.HandlerFunc(s.exportCSVLegacy)))

	app.RunHTTP(addr, mux)
}

func (s *service) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *service) ingestAudit(w http.ResponseWriter, r *http.Request) {
	var req auditEvent
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}
	if req.OrgID == "" || req.Action == "" || req.ObjectType == "" || req.ObjectID == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "org_id, action, object_type, object_id are required")
		return
	}
	if req.Details == nil {
		req.Details = map[string]any{}
	}

	_, err := s.db.Exec(r.Context(), `
		INSERT INTO audit_logs (org_id, project_id, api_key_id, actor_user_id, action, object_type, object_id, details)
		VALUES (
			$1::uuid,
			NULLIF($2, '')::uuid,
			NULLIF($3, '')::uuid,
			NULLIF($4, '')::uuid,
			$5,
			$6,
			$7,
			$8::jsonb
		)
	`, req.OrgID, req.ProjectID, req.APIKeyID, req.ActorUserID, req.Action, req.ObjectType, req.ObjectID, mustJSON(req.Details))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot save audit event")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "stored"})
}

func (s *service) ingestUsage(w http.ResponseWriter, r *http.Request) {
	var req usageEvent
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}
	req.RequestedModel = strings.TrimSpace(req.RequestedModel)
	req.EffectiveModel = strings.TrimSpace(req.EffectiveModel)
	req.Model = strings.TrimSpace(req.Model)

	if req.RequestedModel == "" {
		req.RequestedModel = req.Model
	}
	if req.EffectiveModel == "" {
		if req.Model != "" {
			req.EffectiveModel = req.Model
		} else {
			req.EffectiveModel = req.RequestedModel
		}
	}
	if req.RequestedModel == "" {
		req.RequestedModel = req.EffectiveModel
	}
	if req.Model == "" {
		req.Model = req.EffectiveModel
	}
	// Keep legacy model as backward-compatible alias of effective_model.
	req.Model = req.EffectiveModel

	if req.OrgID == "" || req.ProjectID == "" || req.APIKeyID == "" || req.EffectiveModel == "" || req.TotalTokens <= 0 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "missing required fields")
		return
	}
	_, err := s.db.Exec(r.Context(), `
		INSERT INTO usage_records (org_id, project_id, api_key_id, model, requested_model, effective_model, input_tokens, output_tokens, total_tokens, estimated_cost)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10)
	`, req.OrgID, req.ProjectID, req.APIKeyID, req.Model, req.RequestedModel, req.EffectiveModel, req.InputTokens, req.OutputTokens, req.TotalTokens, req.EstimatedCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot save usage")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "stored"})
}

func (s *service) auditList(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}
	limit := intQuery(r, "limit", 100)
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	query := `
		SELECT id, action, object_type, object_id, actor_user_id::text, details, created_at
		FROM audit_logs
		WHERE org_id = $1::uuid
	`
	args := []any{claims.OrgID}
	switch scope.Scope {
	case "project":
		query += " AND project_id = $2::uuid"
		args = append(args, scope.ProjectID)
	case "key":
		query += " AND api_key_id = $2::uuid"
		args = append(args, scope.APIKeyID)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query audit")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var action, objectType, objectID, actorID string
		var details []byte
		var createdAt time.Time
		if err = rows.Scan(&id, &action, &objectType, &objectID, &actorID, &details, &createdAt); err == nil {
			items = append(items, map[string]any{
				"id":            id,
				"action":        action,
				"object_type":   objectType,
				"object_id":     objectID,
				"actor_user_id": actorID,
				"details":       jsonRaw(details),
				"created_at":    createdAt,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) technicalLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}
	limit := intQuery(r, "limit", 100)
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	apiKeyID := strings.TrimSpace(r.URL.Query().Get("api_key_id"))

	query := `
		SELECT request_id, project_id::text, api_key_id::text, model, status,
		       COALESCE(error_code, ''), retries, fallback_used, COALESCE(fallback_model, ''), input_tokens, output_tokens, created_at
		FROM technical_logs
		WHERE org_id = $1::uuid
	`
	args := []any{claims.OrgID}
	if scope.Scope == "project" {
		args = []any{claims.OrgID, scope.ProjectID}
		query = `
		SELECT request_id, project_id::text, api_key_id::text, model, status,
		       COALESCE(error_code, ''), retries, fallback_used, COALESCE(fallback_model, ''), input_tokens, output_tokens, created_at
		FROM technical_logs
		WHERE org_id = $1::uuid AND project_id = $2::uuid
	`
	} else if scope.Scope == "key" {
		query = `
		SELECT request_id, project_id::text, api_key_id::text, model, status,
		       COALESCE(error_code, ''), retries, fallback_used, COALESCE(fallback_model, ''), input_tokens, output_tokens, created_at
		FROM technical_logs
		WHERE org_id = $1::uuid AND api_key_id = $2::uuid
	`
		args = []any{claims.OrgID, scope.APIKeyID}
	} else if apiKeyID != "" {
		query += " AND api_key_id = $2::uuid"
		args = append(args, apiKeyID)
	}
	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query technical logs")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var requestID, projectID, keyID, model, status, errorCode, fallbackModel string
		var retries int
		var fallbackUsed bool
		var inputTokens, outputTokens int
		var createdAt time.Time
		if err = rows.Scan(
			&requestID,
			&projectID,
			&keyID,
			&model,
			&status,
			&errorCode,
			&retries,
			&fallbackUsed,
			&fallbackModel,
			&inputTokens,
			&outputTokens,
			&createdAt,
		); err == nil {
			items = append(items, map[string]any{
				"request_id":     requestID,
				"project_id":     projectID,
				"api_key_id":     keyID,
				"model":          model,
				"status":         status,
				"error_code":     errorCode,
				"retries":        retries,
				"fallback_used":  fallbackUsed,
				"fallback_model": fallbackModel,
				"input_tokens":   inputTokens,
				"output_tokens":  outputTokens,
				"created_at":     createdAt,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) analytics(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "project"
	}

	selectExpr := "project_id::text"
	switch groupBy {
	case "project":
		selectExpr = "project_id::text"
	case "model":
		selectExpr = "COALESCE(effective_model, model)"
	case "day":
		selectExpr = "to_char(date_trunc('day', created_at), 'YYYY-MM-DD')"
	default:
		httpx.Error(w, http.StatusBadRequest, "validation_error", "group_by must be project/model/day")
		return
	}

	query := fmt.Sprintf(`
		SELECT %s AS grp, SUM(total_tokens), SUM(estimated_cost)
		FROM usage_records
		WHERE org_id = $1::uuid
	`, selectExpr)
	args := []any{claims.OrgID}
	if scope.Scope == "project" {
		query += " AND project_id = $2::uuid"
		args = append(args, scope.ProjectID)
	}
	if scope.Scope == "key" {
		query += " AND api_key_id = $2::uuid"
		args = append(args, scope.APIKeyID)
	}
	query += " GROUP BY grp ORDER BY SUM(estimated_cost) DESC"

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query analytics")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var group string
		var totalTokens int64
		var totalCost float64
		if err = rows.Scan(&group, &totalTokens, &totalCost); err == nil {
			items = append(items, map[string]any{
				"group":        group,
				"total_tokens": totalTokens,
				"total_cost":   totalCost,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"group_by": groupBy, "scope": scope.Scope, "items": items})
}

func (s *service) analyticsTimeseries(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}

	metric := strings.TrimSpace(r.URL.Query().Get("metric"))
	if metric == "" {
		metric = "tokens"
	}
	switch metric {
	case "tokens", "input_tokens", "output_tokens", "cost", "error_rate", "fallback_rate":
	default:
		httpx.Error(w, http.StatusBadRequest, "validation_error", "metric must be tokens|input_tokens|output_tokens|cost|error_rate|fallback_rate")
		return
	}

	requestedBucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	if requestedBucket == "" {
		requestedBucket = "day"
	}
	actualBucket := requestedBucket
	if requestedBucket == "all" {
		actualBucket = "month"
	}
	switch actualBucket {
	case "hour", "day", "week", "month", "year":
	default:
		httpx.Error(w, http.StatusBadRequest, "validation_error", "bucket must be hour|day|week|month|year|all")
		return
	}

	from, err := parseQueryTime(r.URL.Query().Get("from"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid from time")
		return
	}
	to, err := parseQueryTime(r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid to time")
		return
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		if requestedBucket == "all" {
			from = time.Unix(0, 0).UTC()
		} else {
			from = to.AddDate(0, -1, 0)
		}
	}
	if from.After(to) {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "from must be before to")
		return
	}

	points := make([]map[string]any, 0)
	if metric == "tokens" || metric == "input_tokens" || metric == "output_tokens" || metric == "cost" {
		query := fmt.Sprintf(`
			SELECT date_trunc('%s', created_at) AS bucket_start,
			       COALESCE(SUM(input_tokens), 0)::bigint AS input_tokens,
			       COALESCE(SUM(output_tokens), 0)::bigint AS output_tokens,
			       COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens,
			       COALESCE(SUM(estimated_cost), 0)::double precision AS total_cost
			FROM usage_records
			WHERE org_id = $1::uuid
			  AND created_at >= $2
			  AND created_at <= $3
		`, actualBucket)
		args := []any{claims.OrgID, from, to}
		if scope.Scope == "project" {
			query += " AND project_id = $4::uuid"
			args = append(args, scope.ProjectID)
		}
		if scope.Scope == "key" {
			query += " AND api_key_id = $4::uuid"
			args = append(args, scope.APIKeyID)
		}
		query += " GROUP BY bucket_start ORDER BY bucket_start ASC"

		rows, qErr := s.db.Query(r.Context(), query, args...)
		if qErr != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query usage timeseries")
			return
		}
		defer rows.Close()

		for rows.Next() {
			var bucketStart time.Time
			var inputTokens int64
			var outputTokens int64
			var totalTokens int64
			var totalCost float64
			if scanErr := rows.Scan(&bucketStart, &inputTokens, &outputTokens, &totalTokens, &totalCost); scanErr != nil {
				continue
			}
			value := float64(totalTokens)
			if metric == "input_tokens" {
				value = float64(inputTokens)
			}
			if metric == "output_tokens" {
				value = float64(outputTokens)
			}
			if metric == "cost" {
				value = totalCost
			}
			points = append(points, map[string]any{
				"bucket_start": bucketStart.UTC(),
				"value":        value,
			})
		}
	} else {
		query := fmt.Sprintf(`
			SELECT date_trunc('%s', created_at) AS bucket_start,
			       COALESCE(SUM(CASE WHEN status IN ('failed', 'rejected') THEN 1 ELSE 0 END)::double precision / NULLIF(COUNT(*), 0), 0) AS error_rate,
			       COALESCE(SUM(CASE WHEN fallback_used THEN 1 ELSE 0 END)::double precision / NULLIF(COUNT(*), 0), 0) AS fallback_rate
			FROM technical_logs
			WHERE org_id = $1::uuid
			  AND created_at >= $2
			  AND created_at <= $3
		`, actualBucket)
		args := []any{claims.OrgID, from, to}
		if scope.Scope == "project" {
			query += " AND project_id = $4::uuid"
			args = append(args, scope.ProjectID)
		}
		if scope.Scope == "key" {
			query += " AND api_key_id = $4::uuid"
			args = append(args, scope.APIKeyID)
		}
		query += " GROUP BY bucket_start ORDER BY bucket_start ASC"

		rows, qErr := s.db.Query(r.Context(), query, args...)
		if qErr != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query technical timeseries")
			return
		}
		defer rows.Close()

		for rows.Next() {
			var bucketStart time.Time
			var errorRate, fallbackRate float64
			if scanErr := rows.Scan(&bucketStart, &errorRate, &fallbackRate); scanErr != nil {
				continue
			}
			value := errorRate
			if metric == "fallback_rate" {
				value = fallbackRate
			}
			points = append(points, map[string]any{
				"bucket_start": bucketStart.UTC(),
				"value":        value,
			})
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"metric":           metric,
		"scope":            scope.Scope,
		"requested_bucket": requestedBucket,
		"bucket":           actualBucket,
		"from":             from.UTC(),
		"to":               to.UTC(),
		"items":            points,
	})
}

func (s *service) exportCSVLegacy(w http.ResponseWriter, r *http.Request) {
	s.exportCSVUsage(w, r)
}

func (s *service) exportCSVUsage(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}

	query := `
		SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
		       project_id::text,
		       model,
		       COALESCE(requested_model, model) AS requested_model,
		       COALESCE(effective_model, model) AS effective_model,
		       SUM(total_tokens) AS total_tokens,
		       SUM(estimated_cost) AS total_cost
		FROM usage_records
		WHERE org_id = $1::uuid
	`
	args := []any{claims.OrgID}
	if scope.Scope == "project" {
		query += " AND project_id = $2::uuid"
		args = append(args, scope.ProjectID)
	}
	if scope.Scope == "key" {
		query += " AND api_key_id = $2::uuid"
		args = append(args, scope.APIKeyID)
	}
	query += " GROUP BY day, project_id, model, requested_model, effective_model ORDER BY day DESC"

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot build usage report")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=usage-report.csv")
	csvWriter := csv.NewWriter(w)
	_ = csvWriter.Write([]string{"day", "project_id", "model", "requested_model", "effective_model", "total_tokens", "total_cost"})
	for rows.Next() {
		var day, projectID, model, requestedModel, effectiveModel string
		var totalTokens int64
		var totalCost float64
		if err = rows.Scan(&day, &projectID, &model, &requestedModel, &effectiveModel, &totalTokens, &totalCost); err == nil {
			_ = csvWriter.Write([]string{
				day,
				projectID,
				model,
				requestedModel,
				effectiveModel,
				strconv.FormatInt(totalTokens, 10),
				fmt.Sprintf("%.6f", totalCost),
			})
		}
	}
	csvWriter.Flush()
}

func (s *service) exportCSVAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}

	query := `
		SELECT created_at,
		       COALESCE(actor_user_id::text, ''),
		       action,
		       object_type,
		       object_id,
		       details::text
		FROM audit_logs
		WHERE org_id = $1::uuid
	`
	args := []any{claims.OrgID}
	switch scope.Scope {
	case "project":
		query += " AND project_id = $2::uuid"
		args = append(args, scope.ProjectID)
	case "key":
		query += " AND api_key_id = $2::uuid"
		args = append(args, scope.APIKeyID)
	}
	query += " ORDER BY created_at DESC LIMIT 5000"

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot build audit report")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-report.csv")
	csvWriter := csv.NewWriter(w)
	_ = csvWriter.Write([]string{"created_at", "actor_user_id", "action", "object_type", "object_id", "details_json"})
	for rows.Next() {
		var createdAt time.Time
		var actorID, action, objectType, objectID, details string
		if err = rows.Scan(&createdAt, &actorID, &action, &objectType, &objectID, &details); err == nil {
			_ = csvWriter.Write([]string{
				createdAt.UTC().Format(time.RFC3339),
				actorID,
				action,
				objectType,
				objectID,
				details,
			})
		}
	}
	csvWriter.Flush()
}

func (s *service) exportCSVLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	scope, status, code, message := s.resolveScope(r.Context(), r, claims)
	if status != 0 {
		httpx.Error(w, status, code, message)
		return
	}

	query := `
		SELECT created_at,
		       request_id,
		       project_id::text,
		       api_key_id::text,
		       model,
		       status,
		       COALESCE(error_code, ''),
		       retries,
		       fallback_used,
		       COALESCE(fallback_model, ''),
		       input_tokens,
		       output_tokens
		FROM technical_logs
		WHERE org_id = $1::uuid
	`
	args := []any{claims.OrgID}
	if scope.Scope == "project" {
		query += " AND project_id = $2::uuid"
		args = append(args, scope.ProjectID)
	}
	if scope.Scope == "key" {
		query += " AND api_key_id = $2::uuid"
		args = append(args, scope.APIKeyID)
	}
	query += " ORDER BY created_at DESC LIMIT 5000"

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot build logs report")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=technical-logs-report.csv")
	csvWriter := csv.NewWriter(w)
	_ = csvWriter.Write([]string{
		"created_at",
		"request_id",
		"project_id",
		"api_key_id",
		"model",
		"status",
		"error_code",
		"retries",
		"fallback_used",
		"fallback_model",
		"input_tokens",
		"output_tokens",
	})
	for rows.Next() {
		var createdAt time.Time
		var requestID, projectID, keyID, model, status, errorCode, fallbackModel string
		var retries, inputTokens, outputTokens int
		var fallbackUsed bool
		if err = rows.Scan(
			&createdAt,
			&requestID,
			&projectID,
			&keyID,
			&model,
			&status,
			&errorCode,
			&retries,
			&fallbackUsed,
			&fallbackModel,
			&inputTokens,
			&outputTokens,
		); err == nil {
			_ = csvWriter.Write([]string{
				createdAt.UTC().Format(time.RFC3339),
				requestID,
				projectID,
				keyID,
				model,
				status,
				errorCode,
				strconv.Itoa(retries),
				strconv.FormatBool(fallbackUsed),
				fallbackModel,
				strconv.Itoa(inputTokens),
				strconv.Itoa(outputTokens),
			})
		}
	}
	csvWriter.Flush()
}

func (s *service) resolveScope(ctx context.Context, r *http.Request, claims *auth.Claims) (scopeSelection, int, string, string) {
	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if scope == "" {
		scope = "org"
	}
	if scope != "org" && scope != "project" && scope != "key" {
		return scopeSelection{}, http.StatusBadRequest, "validation_error", "scope must be org|project|key"
	}
	if scope == "org" && (claims.Role == "PM" || claims.Role == "Dev") {
		return scopeSelection{}, http.StatusForbidden, "forbidden", "org scope is not allowed for your role"
	}

	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	apiKeyID := strings.TrimSpace(r.URL.Query().Get("api_key_id"))
	out := scopeSelection{Scope: scope}

	if scope == "project" {
		if projectID == "" {
			return scopeSelection{}, http.StatusBadRequest, "validation_error", "project_id is required for project scope"
		}
		allowed, err := s.canAccessProject(ctx, claims, projectID)
		if err != nil {
			return scopeSelection{}, http.StatusInternalServerError, "internal_error", "cannot validate project access"
		}
		if !allowed {
			return scopeSelection{}, http.StatusNotFound, "not_found", "project not found"
		}
		out.ProjectID = projectID
		return out, 0, "", ""
	}

	if scope == "key" {
		if apiKeyID == "" {
			return scopeSelection{}, http.StatusBadRequest, "validation_error", "api_key_id is required for key scope"
		}
		keyProjectID, ok, err := s.keyProject(ctx, claims.OrgID, apiKeyID)
		if err != nil {
			return scopeSelection{}, http.StatusInternalServerError, "internal_error", "cannot validate key scope"
		}
		if !ok {
			return scopeSelection{}, http.StatusNotFound, "not_found", "api key not found"
		}
		if projectID != "" && projectID != keyProjectID {
			return scopeSelection{}, http.StatusNotFound, "not_found", "api key does not belong to project"
		}
		allowed, err := s.canAccessProject(ctx, claims, keyProjectID)
		if err != nil {
			return scopeSelection{}, http.StatusInternalServerError, "internal_error", "cannot validate key project access"
		}
		if !allowed {
			return scopeSelection{}, http.StatusNotFound, "not_found", "project not found"
		}
		out.ProjectID = keyProjectID
		out.APIKeyID = apiKeyID
		return out, 0, "", ""
	}

	if projectID != "" {
		allowed, err := s.canAccessProject(ctx, claims, projectID)
		if err != nil {
			return scopeSelection{}, http.StatusInternalServerError, "internal_error", "cannot validate project access"
		}
		if !allowed {
			return scopeSelection{}, http.StatusNotFound, "not_found", "project not found"
		}
	}
	return out, 0, "", ""
}

func (s *service) canAccessProject(ctx context.Context, claims *auth.Claims, projectID string) (bool, error) {
	if claims.Role == "Admin" || claims.Role == "Finance" {
		var exists bool
		err := s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM projects
				WHERE id = $1::uuid
				  AND org_id = $2::uuid
				  AND deleted_at IS NULL
			)
		`, projectID, claims.OrgID).Scan(&exists)
		return exists, err
	}

	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM projects p
			JOIN project_members pm ON pm.project_id = p.id
			WHERE p.id = $1::uuid
			  AND p.org_id = $2::uuid
			  AND p.deleted_at IS NULL
			  AND pm.user_id = $3::uuid
		)
	`, projectID, claims.OrgID, claims.UserID).Scan(&exists)
	return exists, err
}

func (s *service) keyProject(ctx context.Context, orgID, apiKeyID string) (string, bool, error) {
	var projectID string
	err := s.db.QueryRow(ctx, `
		SELECT p.id::text
		FROM api_keys k
		JOIN projects p ON p.id = k.project_id
		WHERE k.id = $1::uuid
		  AND p.org_id = $2::uuid
		  AND p.deleted_at IS NULL
	`, apiKeyID, orgID).Scan(&projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return projectID, true, nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonRaw(v []byte) any {
	var out any
	if len(v) == 0 {
		return map[string]any{}
	}
	if err := json.Unmarshal(v, &out); err != nil {
		return string(v)
	}
	return out
}

func intQuery(r *http.Request, key string, def int) int {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func parseQueryTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func rctx() context.Context { return context.Background() }

func env(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func ensureSchema(ctx context.Context, pg *pgxpool.Pool) error {
	stmts := []string{
		`ALTER TABLE technical_logs ADD COLUMN IF NOT EXISTS fallback_model TEXT`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS project_id UUID`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS api_key_id UUID`,
		`CREATE INDEX IF NOT EXISTS audit_logs_org_created_idx ON audit_logs (org_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS audit_logs_org_project_created_idx ON audit_logs (org_id, project_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS audit_logs_org_apikey_created_idx ON audit_logs (org_id, api_key_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS project_members (
			project_id UUID NOT NULL REFERENCES projects(id),
			user_id UUID NOT NULL REFERENCES users(id),
			assigned_by UUID NOT NULL REFERENCES users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (project_id, user_id)
		)`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS requested_model TEXT`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS effective_model TEXT`,
		`UPDATE usage_records SET requested_model = model WHERE requested_model IS NULL OR requested_model = ''`,
		`UPDATE usage_records SET effective_model = model WHERE effective_model IS NULL OR effective_model = ''`,
	}
	for _, stmt := range stmts {
		if _, err := pg.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
