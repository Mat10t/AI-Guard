package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"llm-gateway-mvp/internal/platform/app"
	"llm-gateway-mvp/internal/platform/auth"
	"llm-gateway-mvp/internal/platform/db"
	"llm-gateway-mvp/internal/platform/events"
	"llm-gateway-mvp/internal/platform/httpx"
	"llm-gateway-mvp/internal/platform/util"

	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct {
	db         *pgxpool.Pool
	secret     string
	auditURL   string
	publisher  *events.Publisher
	httpClient *http.Client
}

type projectCreateRequest struct {
	Name string `json:"name"`
}

type projectRoutingUpdateRequest struct {
	FallbackModelID *string `json:"fallback_model_id"`
}

type projectMemberAssignRequest struct {
	UserID string `json:"user_id"`
}

type keyCreateRequest struct {
	Name string `json:"name"`
}

type keyResolveResponse struct {
	KeyID     string `json:"key_id"`
	ProjectID string `json:"project_id"`
	OrgID     string `json:"org_id"`
	Status    string `json:"status"`
}

func main() {
	ctx := context.Background()

	addr := env("HTTP_ADDR", ":8082")
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_gateway?sslmode=disable")
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

	s := &service{
		db:         pg,
		secret:     secret,
		auditURL:   auditURL,
		publisher:  events.NewPublisher(brokers),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.Handle("POST /projects", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.createProject)))
	mux.Handle("GET /projects", auth.Middleware(secret)(http.HandlerFunc(s.listProjects)))
	mux.Handle("GET /projects/{id}", auth.Middleware(secret)(http.HandlerFunc(s.getProject)))
	mux.Handle("DELETE /projects/{id}", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.deleteProject)))
	mux.Handle("GET /projects/{id}/members", auth.Middleware(secret)(http.HandlerFunc(s.listProjectMembers)))
	mux.Handle("POST /projects/{id}/members", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.addProjectMember)))
	mux.Handle("DELETE /projects/{id}/members/{user_id}", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.removeProjectMember)))
	mux.Handle("GET /projects/{id}/routing", auth.Middleware(secret)(http.HandlerFunc(s.getProjectRouting)))
	mux.Handle("PUT /projects/{id}/routing", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.updateProjectRouting)))
	mux.Handle("POST /projects/{id}/keys", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.createKey)))
	mux.Handle("GET /projects/{id}/keys", auth.Middleware(secret)(http.HandlerFunc(s.listKeys)))
	mux.Handle("POST /projects/{id}/keys/{key_id}/revoke", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.revokeKey)))
	mux.HandleFunc("GET /internal/keys/resolve", s.resolveKey)

	app.RunHTTP(addr, mux)
}

func (s *service) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *service) createProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	var req projectCreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "name is required")
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var projectID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO projects (id, org_id, name, created_by)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3::uuid)
		RETURNING id::text
	`, claims.OrgID, req.Name, claims.UserID).Scan(&projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create project")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		INSERT INTO project_members (project_id, user_id, assigned_by, created_at)
		VALUES ($1::uuid, $2::uuid, $2::uuid, NOW())
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, projectID, claims.UserID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot assign creator to project")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot commit project create")
		return
	}

	s.publishEvent(r.Context(), "project.created", projectID, map[string]any{
		"project_id": projectID,
		"org_id":     claims.OrgID,
		"actor":      claims.UserID,
	})
	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"actor_user_id": claims.UserID,
		"action":        "project.created",
		"object_type":   "project",
		"object_id":     projectID,
		"details":       map[string]any{"name": req.Name},
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{"id": projectID, "name": req.Name})
}

func (s *service) listProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	query := `
		SELECT p.id::text, p.name, p.created_at
		FROM projects p
		WHERE p.org_id = $1::uuid
		  AND p.deleted_at IS NULL
	`
	args := []any{claims.OrgID}
	if claims.Role == "PM" || claims.Role == "Dev" {
		query += `
		  AND EXISTS (
		    SELECT 1
		    FROM project_members pm
		    WHERE pm.project_id = p.id
		      AND pm.user_id = $2::uuid
		  )
		`
		args = append(args, claims.UserID)
	}
	query += " ORDER BY p.created_at DESC"

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot list projects")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, name string
		var createdAt time.Time
		if err = rows.Scan(&id, &name, &createdAt); err == nil {
			items = append(items, map[string]any{"id": id, "name": name, "created_at": createdAt})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) getProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := r.PathValue("id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	var name string
	var createdAt time.Time
	err = s.db.QueryRow(r.Context(), `
		SELECT name, created_at
		FROM projects
		WHERE id = $1::uuid AND org_id = $2::uuid AND deleted_at IS NULL
	`, id, claims.OrgID).Scan(&name, &createdAt)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "name": name, "created_at": createdAt})
}

func (s *service) deleteProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := r.PathValue("id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	res, err := s.db.Exec(r.Context(), `
		UPDATE projects
		SET deleted_at = NOW()
		WHERE id = $1::uuid AND org_id = $2::uuid AND deleted_at IS NULL
	`, id, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot delete project")
		return
	}
	if res.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    id,
		"actor_user_id": claims.UserID,
		"action":        "project.deleted",
		"object_type":   "project",
		"object_id":     id,
		"details":       map[string]any{},
	})

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *service) getProjectRouting(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")

	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	var fallbackModelID *string
	if err := s.db.QueryRow(r.Context(), `
		SELECT fallback_model_id
		FROM project_routing_settings
		WHERE project_id = $1::uuid
	`, projectID).Scan(&fallbackModelID); err != nil {
		fallbackModelID = nil
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"project_id":        projectID,
		"fallback_model_id": fallbackModelID,
	})
}

func (s *service) updateProjectRouting(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")

	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	var req projectRoutingUpdateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}

	var fallbackModelID *string
	if req.FallbackModelID != nil {
		trimmed := strings.TrimSpace(*req.FallbackModelID)
		if trimmed != "" {
			var exists bool
			if err := s.db.QueryRow(r.Context(), `
				SELECT EXISTS(SELECT 1 FROM provider_models WHERE id = $1)
			`, trimmed).Scan(&exists); err != nil {
				httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate fallback model")
				return
			}
			if !exists {
				httpx.Error(w, http.StatusBadRequest, "validation_error", "fallback model not found")
				return
			}
			fallbackModelID = &trimmed
		}
	}

	if fallbackModelID == nil {
		if _, err := s.db.Exec(r.Context(), `
			DELETE FROM project_routing_settings
			WHERE project_id = $1::uuid
		`, projectID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot clear project routing")
			return
		}
	} else {
		if _, err := s.db.Exec(r.Context(), `
			INSERT INTO project_routing_settings (project_id, fallback_model_id, updated_by, updated_at)
			VALUES ($1::uuid, $2, $3::uuid, NOW())
			ON CONFLICT (project_id)
			DO UPDATE SET
				fallback_model_id = EXCLUDED.fallback_model_id,
				updated_by = EXCLUDED.updated_by,
				updated_at = NOW()
		`, projectID, *fallbackModelID, claims.UserID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot save project routing")
			return
		}
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"actor_user_id": claims.UserID,
		"action":        "project.routing.updated",
		"object_type":   "project",
		"object_id":     projectID,
		"details": map[string]any{
			"fallback_model_id": fallbackModelID,
		},
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"project_id":        projectID,
		"fallback_model_id": fallbackModelID,
	})
}

func (s *service) createKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	var req keyCreateRequest
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &req); err != nil {
			httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid body")
			return
		}
	}

	plain, err := util.RandomToken(32)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot generate key")
		return
	}
	plain = "sk_" + plain
	prefix := plain
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	keyName := strings.TrimSpace(req.Name)
	if keyName == "" {
		keyName = "Key " + prefix
	}

	var keyID string
	err = s.db.QueryRow(r.Context(), `
		INSERT INTO api_keys (id, project_id, key_hash, key_prefix, key_value, status, name, created_by, created_at, revoked_at)
		SELECT gen_random_uuid(), p.id, $1, $2, $3, 'active', $4, $5::uuid, NOW(), NULL
		FROM projects p
		WHERE p.id = $6::uuid AND p.org_id = $7::uuid AND p.deleted_at IS NULL
		RETURNING id::text
	`, util.SHA256(plain), prefix, plain, keyName, claims.UserID, projectID, claims.OrgID).Scan(&keyID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	s.publishEvent(r.Context(), "api_key.created", keyID, map[string]any{
		"key_id":     keyID,
		"project_id": projectID,
		"org_id":     claims.OrgID,
		"actor":      claims.UserID,
	})
	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"api_key_id":    keyID,
		"actor_user_id": claims.UserID,
		"action":        "api_key.created",
		"object_type":   "api_key",
		"object_id":     keyID,
		"details":       map[string]any{"project_id": projectID, "name": keyName},
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":      keyID,
		"name":    keyName,
		"api_key": plain,
		"status":  "active",
		"prefix":  prefix,
	})
}

func (s *service) listKeys(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT
			k.id::text,
			k.name,
			k.status,
			k.key_prefix,
			k.key_value,
			k.created_at,
			k.revoked_at
		FROM api_keys k
		JOIN projects p ON p.id = k.project_id
		WHERE k.project_id = $1::uuid
		  AND p.org_id = $2::uuid
		  AND p.deleted_at IS NULL
		ORDER BY k.created_at DESC
	`, projectID, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot list keys")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var keyID, name, status, prefix string
		var keyValue *string
		var createdAt time.Time
		var revokedAt *time.Time
		if err = rows.Scan(&keyID, &name, &status, &prefix, &keyValue, &createdAt, &revokedAt); err != nil {
			continue
		}
		row := map[string]any{
			"id":         keyID,
			"name":       name,
			"status":     status,
			"prefix":     prefix,
			"api_key":    nil,
			"created_at": createdAt,
		}
		if status == "active" && keyValue != nil && strings.TrimSpace(*keyValue) != "" {
			row["api_key"] = *keyValue
		}
		if revokedAt != nil {
			row["revoked_at"] = *revokedAt
		} else {
			row["revoked_at"] = nil
		}
		items = append(items, row)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) revokeKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	keyID := r.PathValue("key_id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	var revokedKeyID string
	err = s.db.QueryRow(r.Context(), `
		UPDATE api_keys k
		SET status = 'revoked', revoked_at = NOW(), key_value = NULL
		FROM projects p
		WHERE k.project_id = p.id
		  AND p.id = $1::uuid
		  AND p.org_id = $2::uuid
		  AND k.id = $3::uuid
		  AND k.status = 'active'
		RETURNING k.id::text
	`, projectID, claims.OrgID, keyID).Scan(&revokedKeyID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "active key not found for project")
		return
	}

	s.publishEvent(r.Context(), "api_key.revoked", revokedKeyID, map[string]any{
		"key_id":     revokedKeyID,
		"project_id": projectID,
		"org_id":     claims.OrgID,
		"actor":      claims.UserID,
	})
	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"api_key_id":    revokedKeyID,
		"actor_user_id": claims.UserID,
		"action":        "api_key.revoked",
		"object_type":   "api_key",
		"object_id":     revokedKeyID,
		"details":       map[string]any{"project_id": projectID},
	})

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *service) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT u.id::text, u.email, u.role, pm.created_at
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		JOIN projects p ON p.id = pm.project_id
		WHERE pm.project_id = $1::uuid
		  AND p.org_id = $2::uuid
		  AND p.deleted_at IS NULL
		  AND u.deleted_at IS NULL
		ORDER BY pm.created_at ASC
	`, projectID, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot list project members")
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var userID, email, role string
		var createdAt time.Time
		if err = rows.Scan(&userID, &email, &role, &createdAt); err == nil {
			items = append(items, map[string]any{
				"user_id":     userID,
				"email":       email,
				"role":        role,
				"assigned_at": createdAt,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *service) addProjectMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	var req projectMemberAssignRequest
	if err = httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "user_id is required")
		return
	}

	var userRole string
	err = s.db.QueryRow(r.Context(), `
		SELECT role
		FROM users
		WHERE id = $1::uuid
		  AND org_id = $2::uuid
		  AND deleted_at IS NULL
	`, req.UserID, claims.OrgID).Scan(&userRole)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if userRole != "PM" && userRole != "Dev" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "only PM and Dev can be assigned to projects")
		return
	}

	var exists bool
	err = s.db.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1
			FROM project_members
			WHERE project_id = $1::uuid AND user_id = $2::uuid
		)
	`, projectID, req.UserID).Scan(&exists)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot check existing assignment")
		return
	}
	if exists {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "already_assigned"})
		return
	}

	_, err = s.db.Exec(r.Context(), `
		INSERT INTO project_members (project_id, user_id, assigned_by, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, NOW())
	`, projectID, req.UserID, claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot assign member")
		return
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"actor_user_id": claims.UserID,
		"action":        "project.member.assigned",
		"object_type":   "project",
		"object_id":     projectID,
		"details": map[string]any{
			"user_id": req.UserID,
		},
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (s *service) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	projectID := r.PathValue("id")
	userID := strings.TrimSpace(r.PathValue("user_id"))
	if userID == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "user_id is required")
		return
	}
	allowed, err := s.hasProjectAccess(r.Context(), claims, projectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate project access")
		return
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	res, err := s.db.Exec(r.Context(), `
		DELETE FROM project_members pm
		USING users u, projects p
		WHERE pm.project_id = p.id
		  AND pm.user_id = u.id
		  AND pm.project_id = $1::uuid
		  AND pm.user_id = $2::uuid
		  AND p.org_id = $3::uuid
		  AND p.deleted_at IS NULL
		  AND u.deleted_at IS NULL
	`, projectID, userID, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot remove project member")
		return
	}
	if res.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "not_found", "project member not found")
		return
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"project_id":    projectID,
		"actor_user_id": claims.UserID,
		"action":        "project.member.unassigned",
		"object_type":   "project",
		"object_id":     projectID,
		"details": map[string]any{
			"user_id": userID,
		},
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "unassigned"})
}

func (s *service) resolveKey(w http.ResponseWriter, r *http.Request) {
	apiKey := strings.TrimSpace(r.URL.Query().Get("api_key"))
	if apiKey == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "api_key is required")
		return
	}
	hash := util.SHA256(apiKey)

	var out keyResolveResponse
	err := s.db.QueryRow(r.Context(), `
		SELECT k.id::text, k.project_id::text, p.org_id::text, k.status
		FROM api_keys k
		JOIN projects p ON p.id = k.project_id
		WHERE k.key_hash = $1 AND p.deleted_at IS NULL
	`, hash).Scan(&out.KeyID, &out.ProjectID, &out.OrgID, &out.Status)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "key not found")
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *service) publishEvent(ctx context.Context, topic, key string, payload map[string]any) {
	_ = s.publisher.Publish(ctx, topic, key, payload)
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

func (s *service) hasProjectAccess(ctx context.Context, claims *auth.Claims, projectID string) (bool, error) {
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

func env(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func ensureSchema(ctx context.Context, pg *pgxpool.Pool) error {
	stmts := []string{
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_value TEXT`,
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS name TEXT`,
		`UPDATE api_keys SET name = 'Key ' || key_prefix WHERE name IS NULL OR btrim(name) = ''`,
		`ALTER TABLE api_keys ALTER COLUMN name SET NOT NULL`,
		`ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_project_id_key`,
		`CREATE INDEX IF NOT EXISTS api_keys_project_status_idx ON api_keys (project_id, status)`,
		`CREATE INDEX IF NOT EXISTS api_keys_project_created_at_idx ON api_keys (project_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS project_members (
			project_id UUID NOT NULL REFERENCES projects(id),
			user_id UUID NOT NULL REFERENCES users(id),
			assigned_by UUID NOT NULL REFERENCES users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (project_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS project_routing_settings (
			project_id UUID PRIMARY KEY REFERENCES projects(id),
			fallback_model_id TEXT REFERENCES provider_models(id),
			updated_by UUID NOT NULL REFERENCES users(id),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO project_members (project_id, user_id, assigned_by)
		SELECT p.id, u.id, p.created_by
		FROM users u
		JOIN projects p ON p.org_id = u.org_id
		WHERE u.role IN ('PM', 'Dev')
		  AND u.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		ON CONFLICT (project_id, user_id) DO NOTHING`,
	}
	for _, stmt := range stmts {
		if _, err := pg.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
