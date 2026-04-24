package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct {
	db         *pgxpool.Pool
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	auditURL   string
	publisher  *events.Publisher
	httpClient *http.Client
}

type registerRequest struct {
	OrgName  string `json:"org_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type inviteRequest struct {
	Email      string   `json:"email"`
	Role       string   `json:"role"`
	ProjectIDs []string `json:"project_ids"`
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type roleUpdateRequest struct {
	Role string `json:"role"`
}

func main() {
	ctx := context.Background()

	addr := env("HTTP_ADDR", ":8081")
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_gateway?sslmode=disable")
	secret := env("JWT_SECRET", "dev-secret")
	auditURL := env("AUDIT_SERVICE_URL", "http://localhost:8085")
	accessTTL := durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	refreshTTL := durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)
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
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		auditURL:   auditURL,
		publisher:  events.NewPublisher(brokers),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /auth/register", s.register)
	mux.HandleFunc("POST /auth/login", s.login)
	mux.HandleFunc("POST /auth/refresh", s.refresh)
	mux.HandleFunc("POST /auth/logout", s.logout)
	mux.Handle("GET /org/members", auth.Middleware(secret)(http.HandlerFunc(s.members)))
	mux.Handle("POST /org/members/invite", auth.Middleware(secret, "Admin", "PM")(http.HandlerFunc(s.invite)))
	mux.HandleFunc("POST /org/members/accept", s.acceptInvite)
	mux.Handle("PUT /org/members/{id}/role", auth.Middleware(secret, "Admin")(http.HandlerFunc(s.updateRole)))
	mux.Handle("DELETE /org/members/{id}", auth.Middleware(secret, "Admin")(http.HandlerFunc(s.deleteMember)))

	app.RunHTTP(addr, mux)
}

func (s *service) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *service) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	req.OrgName = strings.TrimSpace(req.OrgName)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.OrgName == "" || req.Email == "" || len(req.Password) < 8 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "org_name, email and password(>=8) are required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot hash password")
		return
	}

	var userID, orgID string
	err = s.db.QueryRow(r.Context(), `
		WITH new_org AS (
			INSERT INTO organizations (id, name)
			VALUES (gen_random_uuid(), $1)
			RETURNING id
		),
		new_user AS (
			INSERT INTO users (id, org_id, email, password_hash, role)
			SELECT gen_random_uuid(), id, $2, $3, 'Admin'
			FROM new_org
			RETURNING id, org_id
		)
		SELECT new_user.id::text, new_user.org_id::text
		FROM new_user
	`, req.OrgName, req.Email, hash).Scan(&userID, &orgID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.Error(w, http.StatusConflict, "conflict", "email already exists")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create organization")
		return
	}

	access, refresh, err := s.issueSession(r.Context(), userID, orgID, "Admin")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create session")
		return
	}
	s.setRefreshCookie(w, refresh)

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        orgID,
		"actor_user_id": userID,
		"action":        "org.registered",
		"object_type":   "organization",
		"object_id":     orgID,
		"details":       map[string]any{"email": req.Email},
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"access_token": access,
		"user_id":      userID,
		"org_id":       orgID,
		"role":         "Admin",
	})
}

func (s *service) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var userID, orgID, role, hash string
	err := s.db.QueryRow(r.Context(), `
		SELECT id::text, org_id::text, role, password_hash
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, req.Email).Scan(&userID, &orgID, &role, &hash)
	if err != nil || !auth.VerifyPassword(hash, req.Password) {
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}

	access, refresh, err := s.issueSession(r.Context(), userID, orgID, role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create session")
		return
	}
	s.setRefreshCookie(w, refresh)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"user_id":      userID,
		"org_id":       orgID,
		"role":         role,
	})
}

func (s *service) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		httpx.Error(w, http.StatusUnauthorized, "missing_refresh", "refresh token is required")
		return
	}
	oldHash := util.SHA256(cookie.Value)

	var sessionID, userID, orgID, role string
	err = s.db.QueryRow(r.Context(), `
		SELECT s.id::text, u.id::text, u.org_id::text, u.role
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		  AND u.deleted_at IS NULL
	`, oldHash).Scan(&sessionID, &userID, &orgID, &role)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid_refresh", "refresh token is invalid")
		return
	}

	_, _ = s.db.Exec(r.Context(), `UPDATE sessions SET revoked_at = NOW() WHERE id = $1::uuid`, sessionID)
	access, refresh, err := s.issueSession(r.Context(), userID, orgID, role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot rotate session")
		return
	}
	s.setRefreshCookie(w, refresh)

	httpx.JSON(w, http.StatusOK, map[string]any{"access_token": access})
}

func (s *service) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		h := util.SHA256(cookie.Value)
		_, _ = s.db.Exec(r.Context(), `UPDATE sessions SET revoked_at = NOW() WHERE refresh_hash = $1`, h)
	}
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (s *service) members(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT id::text, email, role, created_at
		FROM users
		WHERE org_id = $1::uuid
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot query members")
		return
	}
	defer rows.Close()

	resp := make([]map[string]any, 0)
	for rows.Next() {
		var id, email, role string
		var createdAt time.Time
		if err = rows.Scan(&id, &email, &role, &createdAt); err == nil {
			resp = append(resp, map[string]any{"id": id, "email": email, "role": role, "created_at": createdAt})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": resp})
}

func (s *service) invite(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	var req inviteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.TrimSpace(req.Role)
	if req.Email == "" || !isAllowedRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "email and valid role are required")
		return
	}
	req.ProjectIDs = normalizeUUIDList(req.ProjectIDs)
	if roleRequiresProjects(req.Role) && len(req.ProjectIDs) == 0 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "project_ids are required for PM and Dev invites")
		return
	}
	if len(req.ProjectIDs) > 0 {
		for _, projectID := range req.ProjectIDs {
			var exists bool
			err := s.db.QueryRow(r.Context(), `
				SELECT EXISTS(
					SELECT 1
					FROM projects
					WHERE id = $1::uuid
					  AND org_id = $2::uuid
					  AND deleted_at IS NULL
				)
			`, projectID, claims.OrgID).Scan(&exists)
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate projects")
				return
			}
			if !exists {
				httpx.Error(w, http.StatusBadRequest, "validation_error", "all project_ids must belong to your organization")
				return
			}
			if claims.Role == "PM" {
				var assigned bool
				err = s.db.QueryRow(r.Context(), `
					SELECT EXISTS(
						SELECT 1
						FROM project_members pm
						JOIN projects p ON p.id = pm.project_id
						WHERE pm.user_id = $1::uuid
						  AND pm.project_id = $2::uuid
						  AND p.org_id = $3::uuid
						  AND p.deleted_at IS NULL
					)
				`, claims.UserID, projectID, claims.OrgID).Scan(&assigned)
				if err != nil {
					httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot validate inviter project access")
					return
				}
				if !assigned {
					httpx.Error(w, http.StatusForbidden, "forbidden", "PM can invite only to assigned projects")
					return
				}
			}
		}
	}

	token, err := util.RandomToken(24)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create invite token")
		return
	}

	var projectIDs any
	if len(req.ProjectIDs) > 0 {
		projectIDs = req.ProjectIDs
	}
	_, err = s.db.Exec(r.Context(), `
		INSERT INTO invitations (id, org_id, email, role, project_ids, token_hash, expires_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, $4::uuid[], $5, NOW() + INTERVAL '72 hours')
	`, claims.OrgID, req.Email, req.Role, projectIDs, util.SHA256(token))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create invitation")
		return
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"actor_user_id": claims.UserID,
		"action":        "member.invited",
		"object_type":   "invitation",
		"object_id":     req.Email,
		"details":       map[string]any{"role": req.Role, "project_ids": req.ProjectIDs},
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"invitation_token": token,
		"email":            req.Email,
		"role":             req.Role,
		"project_ids":      req.ProjectIDs,
	})
}

func (s *service) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var req acceptInviteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Token == "" || len(req.Password) < 8 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "token and password(>=8) are required")
		return
	}

	var orgID, email, role string
	var projectIDs []string
	err := s.db.QueryRow(r.Context(), `
		SELECT org_id::text, email, role, COALESCE(project_ids, '{}'::uuid[])::text[]
		FROM invitations
		WHERE token_hash = $1
		  AND accepted_at IS NULL
		  AND expires_at > NOW()
	`, util.SHA256(req.Token)).Scan(&orgID, &email, &role, &projectIDs)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "invitation not found or expired")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot hash password")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var userID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO users (id, org_id, email, password_hash, role)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, $4)
		RETURNING id::text
	`, orgID, email, hash, role).Scan(&userID)
	if err != nil {
		httpx.Error(w, http.StatusConflict, "conflict", "user already exists")
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE invitations SET accepted_at = NOW()
		WHERE token_hash = $1
	`, util.SHA256(req.Token))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot update invitation")
		return
	}
	if roleRequiresProjects(role) && len(projectIDs) > 0 {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO project_members (project_id, user_id, assigned_by, created_at)
			SELECT p.id, $1::uuid, $1::uuid, NOW()
			FROM projects p
			WHERE p.id = ANY($2::uuid[])
			  AND p.org_id = $3::uuid
			  AND p.deleted_at IS NULL
			ON CONFLICT (project_id, user_id) DO NOTHING
		`, userID, projectIDs, orgID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create project assignments")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot commit invitation acceptance")
		return
	}

	access, refresh, err := s.issueSession(r.Context(), userID, orgID, role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot create session")
		return
	}
	s.setRefreshCookie(w, refresh)

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        orgID,
		"actor_user_id": userID,
		"action":        "member.accepted",
		"object_type":   "user",
		"object_id":     userID,
		"details":       map[string]any{"email": email, "role": role, "project_ids": projectIDs},
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"access_token": access,
		"user_id":      userID,
		"org_id":       orgID,
		"role":         role,
		"project_ids":  projectIDs,
	})
}

func (s *service) updateRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "member id is required")
		return
	}
	var req roleUpdateRequest
	var err error
	if err = httpx.Decode(r, &req); err != nil || !isAllowedRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "valid role is required")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var currentRole string
	err = tx.QueryRow(r.Context(), `
		SELECT role
		FROM users
		WHERE id = $1::uuid AND org_id = $2::uuid AND deleted_at IS NULL
		FOR UPDATE
	`, id, claims.OrgID).Scan(&currentRole)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot load user")
		return
	}

	if currentRole == "Admin" && req.Role != "Admin" {
		var adminCount int
		err = tx.QueryRow(r.Context(), `
			SELECT COUNT(*)
			FROM users
			WHERE org_id = $1::uuid AND role = 'Admin' AND deleted_at IS NULL
		`, claims.OrgID).Scan(&adminCount)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot count admins")
			return
		}
		if adminCount <= 1 {
			httpx.Error(w, http.StatusConflict, "conflict", "cannot demote last admin")
			return
		}
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE users
		SET role = $1
		WHERE id = $2::uuid AND org_id = $3::uuid AND deleted_at IS NULL
	`, req.Role, id, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot update role")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot commit role update")
		return
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"actor_user_id": claims.UserID,
		"action":        "member.role.updated",
		"object_type":   "user",
		"object_id":     id,
		"details":       map[string]any{"role": req.Role},
	})

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *service) deleteMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "member id is required")
		return
	}
	if id == claims.UserID {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "cannot delete yourself")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var targetRole string
	err = tx.QueryRow(r.Context(), `
		SELECT role
		FROM users
		WHERE id = $1::uuid AND org_id = $2::uuid AND deleted_at IS NULL
		FOR UPDATE
	`, id, claims.OrgID).Scan(&targetRole)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot load user")
		return
	}

	if targetRole == "Admin" {
		var adminCount int
		err = tx.QueryRow(r.Context(), `
			SELECT COUNT(*)
			FROM users
			WHERE org_id = $1::uuid AND role = 'Admin' AND deleted_at IS NULL
		`, claims.OrgID).Scan(&adminCount)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot count admins")
			return
		}
		if adminCount <= 1 {
			httpx.Error(w, http.StatusConflict, "conflict", "cannot delete last admin")
			return
		}
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE users
		SET deleted_at = NOW()
		WHERE id = $1::uuid AND org_id = $2::uuid AND deleted_at IS NULL
	`, id, claims.OrgID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot delete user")
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE user_id = $1::uuid
		  AND revoked_at IS NULL
	`, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot revoke sessions")
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "cannot commit delete")
		return
	}

	s.publishAudit(r.Context(), map[string]any{
		"org_id":        claims.OrgID,
		"actor_user_id": claims.UserID,
		"action":        "member.deleted",
		"object_type":   "user",
		"object_id":     id,
		"details":       map[string]any{},
	})

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *service) issueSession(ctx context.Context, userID, orgID, role string) (string, string, error) {
	access, err := auth.NewAccessToken(s.secret, userID, orgID, role, s.accessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err := util.RandomToken(48)
	if err != nil {
		return "", "", err
	}
	refreshHash := util.SHA256(refresh)
	seconds := int64(s.refreshTTL.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO sessions (id, user_id, refresh_hash, expires_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, NOW() + ($3 * interval '1 second'))
	`, userID, refreshHash, seconds)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *service) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.refreshTTL.Seconds()),
	})
}

func (s *service) publishAudit(ctx context.Context, payload map[string]any) {
	if s.auditURL != "" {
		b, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.auditURL+"/internal/audit/event", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		_, _ = s.httpClient.Do(req)
	}
	_ = s.publisher.Publish(ctx, "audit.event.created", payload["object_id"].(string), payload)
}

func isAllowedRole(role string) bool {
	switch role {
	case "Admin", "PM", "Dev", "Finance":
		return true
	default:
		return false
	}
}

func roleRequiresProjects(role string) bool {
	return role == "PM" || role == "Dev"
}

func normalizeUUIDList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func env(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func durationEnv(k string, def time.Duration) time.Duration {
	raw := os.Getenv(k)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}

func ensureSchema(ctx context.Context, pg *pgxpool.Pool) error {
	queries := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ`,
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_email_active_unique_idx ON users (email) WHERE deleted_at IS NULL`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS project_ids UUID[]`,
		`CREATE TABLE IF NOT EXISTS project_members (
			project_id UUID NOT NULL REFERENCES projects(id),
			user_id UUID NOT NULL REFERENCES users(id),
			assigned_by UUID NOT NULL REFERENCES users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (project_id, user_id)
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
	for _, q := range queries {
		if _, err := pg.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
