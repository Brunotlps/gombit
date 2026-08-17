package auth

import (
	"testing"
	"time"
)

func TestSignAndParseAccessToken(t *testing.T) {
	secret := []byte("test-jwt-secret-32-bytes-minimum!")
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	user := User{ID: 7, Email: "ada@example.com"}

	token, err := signAccessToken(secret, user, 1, time.Minute, now)
	if err != nil {
		t.Fatalf("signAccessToken() error = %v", err)
	}
	claims, err := parseAccessToken(secret, token, now)
	if err != nil {
		t.Fatalf("parseAccessToken() error = %v", err)
	}
	if claims.Email != user.Email {
		t.Fatalf("email = %q, want %q", claims.Email, user.Email)
	}
	id, err := userIDFromSubject(claims.Subject)
	if err != nil {
		t.Fatalf("userIDFromSubject() error = %v", err)
	}
	if id != user.ID {
		t.Fatalf("id = %d, want %d", id, user.ID)
	}
}

func TestParseAccessTokenRejectsExpired(t *testing.T) {
	secret := []byte("test-jwt-secret-32-bytes-minimum!")
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	user := User{ID: 1, Email: "ada@example.com"}
	token, err := signAccessToken(secret, user, 1, time.Second, now)
	if err != nil {
		t.Fatalf("signAccessToken() error = %v", err)
	}
	_, err = parseAccessToken(secret, token, now.Add(2*time.Second))
	if err == nil {
		t.Fatal("parseAccessToken() error = nil, want expired")
	}
}

func TestParseAccessTokenRejectsWrongSecret(t *testing.T) {
	now := time.Now()
	token, err := signAccessToken([]byte("test-jwt-secret-32-bytes-minimum!"), User{ID: 1, Email: "a@b.c"}, 1, time.Minute, now)
	if err != nil {
		t.Fatalf("signAccessToken() error = %v", err)
	}
	_, err = parseAccessToken([]byte("other-jwt-secret-32-bytes-minimum"), token, now)
	if err == nil {
		t.Fatal("parseAccessToken() error = nil, want invalid")
	}
}

func TestRefreshTokenHashIsDeterministic(t *testing.T) {
	raw, err := newRefreshToken()
	if err != nil {
		t.Fatalf("newRefreshToken() error = %v", err)
	}
	first := hashRefreshToken(raw)
	second := hashRefreshToken(raw)
	if first != second {
		t.Fatal("hashRefreshToken is not deterministic")
	}
	if first == hashRefreshToken(raw+"x") {
		t.Fatal("hashRefreshToken collided across distinct tokens")
	}
}
