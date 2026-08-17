package auth

import (
	"errors"
	"strings"
)

var (
	errPasswordRequired   = errors.New("password is required")
	errPasswordTooLong    = errors.New("password must be at most 72 bytes")
	errInvalidCredentials = errors.New("invalid email or password")
	errEmailTaken         = errors.New("email already registered")
	errUserNotFound       = errors.New("user not found")
	errRefreshReuse       = errors.New("refresh token reuse detected")
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
