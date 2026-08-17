package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"gorm.io/gorm"
)

// Service is the runtime auth implementation: users, password hashes,
// access JWTs, and rotating refresh tokens.
type Service struct {
	db        *gorm.DB
	cfg       config.AuthConfig
	secret    []byte
	hasher    Hasher
	clock     Clock
	dummyHash string
}

// NewService constructs a Service. cfg.Auth.JWTSecret must be set.
func NewService(db *gorm.DB, cfg config.Config) (*Service, error) {
	if db == nil {
		return nil, errors.New("auth: nil database")
	}
	if !cfg.Auth.Enabled() {
		return nil, errors.New("auth: JWT secret is not set")
	}
	if cfg.Auth.AccessTokenTTL <= 0 || cfg.Auth.RefreshTokenTTL <= 0 {
		return nil, errors.New("auth: token TTLs must be positive")
	}
	s := &Service{
		db:     db,
		cfg:    cfg.Auth,
		secret: []byte(cfg.Auth.JWTSecret),
		hasher: newBcryptHasher(cfg.Auth.BcryptCost),
		clock:  systemClock{},
	}
	return s, nil
}

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now()
	}
	return s.clock.Now()
}

// Register creates a user. Duplicate emails return errEmailTaken.
func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	if email == "" {
		return User{}, errInvalidCredentials
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, err
	}
	user := User{Email: email, PasswordHash: hash}
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		if isUniqueViolation(err) {
			return User{}, errEmailTaken
		}
		return User{}, err
	}
	return user, nil
}

// Authenticate verifies email and password.
func (s *Service) Authenticate(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	var user User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		s.compareDummy(password)
		return User{}, errInvalidCredentials
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return User{}, errInvalidCredentials
	}
	return user, nil
}

// IssueTokens returns a new access JWT and a rotating refresh token.
func (s *Service) IssueTokens(ctx context.Context, user User) (TokenPair, error) {
	now := s.now()
	raw, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	row := RefreshToken{
		UserID:    user.ID,
		TokenHash: hashRefreshToken(raw),
		ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return TokenPair{}, err
	}
	access, err := signAccessToken(s.secret, user, row.ID, s.cfg.AccessTokenTTL, now)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: raw,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// RotateRefresh validates the current refresh token, revokes it, and issues a
// new pair. Reuse of a revoked token revokes the user's remaining tokens.
func (s *Service) RotateRefresh(ctx context.Context, raw string) (TokenPair, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return TokenPair{}, errInvalidRefreshToken
	}
	now := s.now()
	var row RefreshToken
	err := s.db.WithContext(ctx).Where("token_hash = ?", hashRefreshToken(raw)).First(&row).Error
	if err != nil {
		return TokenPair{}, errInvalidRefreshToken
	}
	if row.RevokedAt != nil {
		_ = s.revokeAll(ctx, row.UserID, now)
		return TokenPair{}, errRefreshReuse
	}
	if !row.ExpiresAt.After(now) {
		return TokenPair{}, errInvalidRefreshToken
	}

	var user User
	if err := s.db.WithContext(ctx).First(&user, row.UserID).Error; err != nil {
		return TokenPair{}, errUserNotFound
	}

	pair, err := s.IssueTokens(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}

	var replacement RefreshToken
	if err := s.db.WithContext(ctx).Where("token_hash = ?", hashRefreshToken(pair.RefreshToken)).First(&replacement).Error; err != nil {
		return TokenPair{}, err
	}
	revoked := now
	row.RevokedAt = &revoked
	row.ReplacedBy = &replacement.ID
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

// RevokeRefresh invalidates one refresh token. Missing/already-revoked tokens
// succeed so logout is idempotent.
func (s *Service) RevokeRefresh(ctx context.Context, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errInvalidRefreshToken
	}
	now := s.now()
	return s.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hashRefreshToken(raw)).
		Update("revoked_at", now).Error
}

// ParseAccess validates a Bearer access JWT and loads the user.
func (s *Service) ParseAccess(ctx context.Context, token string) (User, error) {
	claims, err := parseAccessToken(s.secret, token, s.now())
	if err != nil {
		return User{}, err
	}
	id, err := userIDFromSubject(claims.Subject)
	if err != nil {
		return User{}, err
	}
	var user User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return User{}, errInvalidAccessToken
	}
	var row RefreshToken
	if err := s.db.WithContext(ctx).Where("id = ?", claims.RefreshID).First(&row).Error; err != nil {
		return User{}, errInvalidAccessToken
	}
	if row.UserID != user.ID || row.RevokedAt != nil {
		return User{}, errInvalidAccessToken
	}
	return user, nil
}

func (s *Service) revokeAll(ctx context.Context, userID uint, now time.Time) error {
	return s.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (s *Service) compareDummy(password string) {
	if s.dummyHash == "" {
		hash, err := s.hasher.Hash("timing-dummy-password")
		if err != nil {
			return
		}
		s.dummyHash = hash
	}
	_ = s.hasher.Compare(s.dummyHash, password)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

// TokenPair is the login/refresh success payload (inside D10 data).
type TokenPair struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiJ9..." doc:"Short-lived Bearer access JWT. Hold in memory only."`
	RefreshToken string `json:"refresh_token" example:"aa11..." doc:"Opaque rotating refresh token. Hold in memory only; never web storage."`
	TokenType    string `json:"token_type" example:"Bearer" doc:"Token type; always Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"900" doc:"Access token lifetime in seconds"`
}

// PublicUser is the safe user representation returned by register and /me.
type PublicUser struct {
	ID    uint   `json:"id" example:"1" doc:"User identifier"`
	Email string `json:"email" example:"ada@example.com" format:"email" doc:"Normalized email"`
}

func toPublicUser(user User) PublicUser {
	return PublicUser{ID: user.ID, Email: user.Email}
}
