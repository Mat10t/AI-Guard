
package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// HashPassword & VerifyPassword
// -----------------------------------------------------------------------------

func TestHashPassword(t *testing.T) {
	password := "mySecret123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Error("HashPassword returned empty hash")
	}
	if hash == password {
		t.Error("HashPassword returned raw password as hash")
	}
	if !VerifyPassword(hash, password) {
		t.Error("VerifyPassword failed for correct password")
	}
	if VerifyPassword(hash, "wrong") {
		t.Error("VerifyPassword succeeded for wrong password")
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, _ := HashPassword("correct")
	tests := []struct {
		name     string
		hash     string
		password string
		want     bool
	}{
		{"correct password", hash, "correct", true},
		{"wrong password", hash, "wrong", false},
		{"empty password", hash, "", false},
		{"invalid hash", "invalid", "anything", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyPassword(tt.hash, tt.password); got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// NewAccessToken & ParseAccessToken
// -----------------------------------------------------------------------------

const testSecret = "my-test-secret-key"

func TestNewAccessToken(t *testing.T) {
	userID := "user-123"
	orgID := "org-456"
	role := "admin"
	ttl := 10 * time.Minute

	token, err := NewAccessToken(testSecret, userID, orgID, role, ttl)
	if err != nil {
		t.Fatalf("NewAccessToken failed: %v", err)
	}
	if token == "" {
		t.Error("NewAccessToken returned empty token")
	}

	claims, err := ParseAccessToken(testSecret, token)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.OrgID != orgID {
		t.Errorf("OrgID = %q, want %q", claims.OrgID, orgID)
	}
	if claims.Role != role {
		t.Errorf("Role = %q, want %q", claims.Role, role)
	}
	if claims.ExpiresAt.Time.Before(time.Now().Add(ttl - time.Second)) {
		t.Error("ExpiresAt too early")
	}
}

func TestParseAccessToken(t *testing.T) {
	// Valid token
	validToken, _ := NewAccessToken(testSecret, "u1", "o1", "viewer", time.Hour)

	// Expired token
	expiredToken, _ := NewAccessToken(testSecret, "u2", "o2", "admin", -time.Hour)

	// Wrong secret
	wrongSecretToken, _ := NewAccessToken("different-secret", "u3", "o3", "admin", time.Hour)

	// Malformed token
	malformed := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.malformed"

	tests := []struct {
		name    string
		secret  string
		token   string
		wantErr bool
	}{
		{"valid token", testSecret, validToken, false},
		{"expired token", testSecret, expiredToken, true},
		{"wrong secret", testSecret, wrongSecretToken, true},
		{"malformed token", testSecret, malformed, true},
		{"empty token", testSecret, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ParseAccessToken(tt.secret, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && claims == nil {
				t.Error("ParseAccessToken returned nil claims for valid token")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ExtractBearerToken
// -----------------------------------------------------------------------------

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   string
	}{
		{
			name:   "valid bearer token",
			header: http.Header{"Authorization": []string{"Bearer abc123"}},
			want:   "abc123",
		},
		{
			name:   "lowercase bearer",
			header: http.Header{"Authorization": []string{"bearer xyz"}},
			want:   "xyz",
		},
		{
			name:   "missing Authorization header",
			header: http.Header{},
			want:   "",
		},
		{
			name:   "empty Authorization",
			header: http.Header{"Authorization": []string{""}},
			want:   "",
		},
		{
			name:   "no Bearer scheme",
			header: http.Header{"Authorization": []string{"Basic dXNlcjpwYXNz"}},
			want:   "",
		},
		{
			name:   "only Bearer without token",
			header: http.Header{"Authorization": []string{"Bearer"}},
			want:   "",
		},
		{
			name:   "multiple spaces",
			header: http.Header{"Authorization": []string{"Bearer   token-with-spaces"}},
			want:   "  token-with-spaces", // SplitN yields ["Bearer", "  token-with-spaces"]
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: tt.header}
			if got := ExtractBearerToken(req); got != tt.want {
				t.Errorf("ExtractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Middleware
// -----------------------------------------------------------------------------

func TestMiddleware(t *testing.T) {
	// Helper to create a valid token for a given role
	validTokenForRole := func(role string) string {
		token, _ := NewAccessToken(testSecret, "u1", "o1", role, time.Minute)
		return token
	}

	// A handler that asserts claims are present in context
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "claims missing", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(claims.Role))
	})

	tests := []struct {
		name           string
		roles          []string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "no roles required, valid token",
			roles:          []string{},
			token:          validTokenForRole("any"),
			expectedStatus: http.StatusOK,
			expectedBody:   "any",
		},
		{
			name:           "no roles required, missing token",
			roles:          []string{},
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "missing token\n",
		},
		{
			name:           "role allowed",
			roles:          []string{"admin", "user"},
			token:          validTokenForRole("user"),
			expectedStatus: http.StatusOK,
			expectedBody:   "user",
		},
		{
			name:           "role not allowed",
			roles:          []string{"admin"},
			token:          validTokenForRole("user"),
			expectedStatus: http.StatusForbidden,
			expectedBody:   "forbidden\n",
		},
		{
			name:           "invalid token",
			roles:          []string{"admin"},
			token:          "invalid.token.string",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid token\n",
		},
		{
			name:           "empty token string",
			roles:          []string{"admin"},
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "missing token\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Middleware(testSecret, tt.roles...)
			handler := mw(successHandler)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
			if rr.Body.String() != tt.expectedBody {
				t.Errorf("body = %q, want %q", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestMiddleware_ClaimsInContext verifies that claims are correctly injected.
func TestMiddleware_ClaimsInContext(t *testing.T) {
	expectedClaims := &Claims{
		UserID: "test-user",
		OrgID:  "test-org",
		Role:   "editor",
	}
	token, err := NewAccessToken(testSecret, expectedClaims.UserID, expectedClaims.OrgID, expectedClaims.Role, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	var capturedClaims *Claims
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("claims not found in context")
		}
		capturedClaims = c
	})

	mw := Middleware(testSecret)
	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}()))

	if capturedClaims == nil {
		t.Fatal("claims were not captured")
	}
	if capturedClaims.UserID != expectedClaims.UserID {
		t.Errorf("UserID = %q, want %q", capturedClaims.UserID, expectedClaims.UserID)
	}
	if capturedClaims.OrgID != expectedClaims.OrgID {
		t.Errorf("OrgID = %q, want %q", capturedClaims.OrgID, expectedClaims.OrgID)
	}
	if capturedClaims.Role != expectedClaims.Role {
		t.Errorf("Role = %q, want %q", capturedClaims.Role, expectedClaims.Role)
	}
}

// -----------------------------------------------------------------------------
// ClaimsFromContext
// -----------------------------------------------------------------------------

func TestClaimsFromContext(t *testing.T) {
	ctx := context.Background()
	// No claims
	if _, ok := ClaimsFromContext(ctx); ok {
		t.Error("ClaimsFromContext on empty context returned ok=true")
	}

	// With claims
	claims := &Claims{UserID: "me"}
	ctxWithClaims := context.WithValue(ctx, ContextClaimsKey, claims)
	c, ok := ClaimsFromContext(ctxWithClaims)
	if !ok {
		t.Error("ClaimsFromContext on context with claims returned ok=false")
	}
	if c != claims {
		t.Error("returned claims pointer differs from stored")
	}

	// Wrong type stored
	ctxWrong := context.WithValue(ctx, ContextClaimsKey, "string")
	if _, ok := ClaimsFromContext(ctxWrong); ok {
		t.Error("ClaimsFromContext with wrong type returned ok=true")
	}
}
