package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
	"golang.org/x/crypto/bcrypt"
)

func TestRotateRefreshKeepsNewAccessValid(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:" + filepath.Join(t.TempDir(), "auth.db") + "?cache=shared&_fk=1",
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.Auth.JWTSecret = "test-jwt-secret-32-bytes-minimum!"
	cfg.Auth.BcryptCost = bcrypt.MinCost
	cfg.Auth.AccessTokenTTL = time.Minute
	cfg.Auth.RefreshTokenTTL = time.Hour
	svc, err := NewService(db.DB, cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	ctx := context.Background()
	user, err := svc.Register(ctx, "ada@example.com", "correct-horse")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	first, err := svc.IssueTokens(ctx, user)
	if err != nil {
		t.Fatalf("IssueTokens: %v", err)
	}
	if _, err := svc.ParseAccess(ctx, first.AccessToken); err != nil {
		t.Fatalf("ParseAccess(first): %v", err)
	}

	second, err := svc.RotateRefresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("RotateRefresh: %v", err)
	}

	var rows []RefreshToken
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	for _, row := range rows {
		t.Logf("token id=%d user=%d revoked=%v replaced=%v", row.ID, row.UserID, row.RevokedAt, row.ReplacedBy)
	}

	if _, err := svc.ParseAccess(ctx, second.AccessToken); err != nil {
		t.Fatalf("ParseAccess(second): %v", err)
	}
	if _, err := svc.RotateRefresh(ctx, first.RefreshToken); err == nil {
		t.Fatal("old refresh should be invalid")
	}
}
