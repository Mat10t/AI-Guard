package auth

import (
	"testing"
	"time"
)

func TestAccessTokenRoundtrip(t *testing.T) {
	secret := "test-secret"
	tok, err := NewAccessToken(secret, "u1", "o1", "Admin", time.Hour)
	if err != nil {
		t.Fatalf("token create err: %v", err)
	}
	claims, err := ParseAccessToken(secret, tok)
	if err != nil {
		t.Fatalf("token parse err: %v", err)
	}
	if claims.UserID != "u1" || claims.OrgID != "o1" || claims.Role != "Admin" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestPasswordHashVerify(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("hash err: %v", err)
	}
	if !VerifyPassword(hash, "password123") {
		t.Fatalf("expected valid password")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatalf("expected invalid password")
	}
}
