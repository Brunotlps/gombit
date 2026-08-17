package config

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactDSNHidesURLPassword(t *testing.T) {
	t.Parallel()

	dsn := "postgres://gombit:super-secret@127.0.0.1:5432/gombit?sslmode=disable" // #nosec G101 -- fake local test DSN.
	got := RedactDSN(dsn)
	if strings.Contains(got, "super-secret") {
		t.Fatalf("RedactDSN() = %q, still contains password", got)
	}
	if !strings.Contains(got, RedactedSecret) {
		t.Fatalf("RedactDSN() = %q, want %s placeholder", got, RedactedSecret)
	}
	if !strings.Contains(got, "gombit") || !strings.Contains(got, "127.0.0.1") {
		t.Fatalf("RedactDSN() = %q, want username and host preserved", got)
	}
}

func TestRedactDSNHidesMySQLUserinfo(t *testing.T) {
	t.Parallel()

	dsn := "gombit:mysql-secret@tcp(127.0.0.1:3306)/gombit?parseTime=true"
	got := RedactDSN(dsn)
	if strings.Contains(got, "mysql-secret") {
		t.Fatalf("RedactDSN() = %q, still contains password", got)
	}
	if got != "gombit:"+RedactedSecret+"@tcp(127.0.0.1:3306)/gombit?parseTime=true" {
		t.Fatalf("RedactDSN() = %q, want mysql userinfo redacted", got)
	}
}

func TestRedactDSNLeavesSQLiteFilePath(t *testing.T) {
	t.Parallel()

	dsn := "file:gombit.db?cache=shared&_fk=1"
	if got := RedactDSN(dsn); got != dsn {
		t.Fatalf("RedactDSN() = %q, want unchanged sqlite DSN", got)
	}
}

func TestRedactDSNHidesQueryPassword(t *testing.T) {
	t.Parallel()

	dsn := "file:gombit.db?cache=shared&password=query-secret"
	got := RedactDSN(dsn)
	if strings.Contains(got, "query-secret") {
		t.Fatalf("RedactDSN() = %q, still contains query password", got)
	}
	if !strings.Contains(got, "password="+RedactedSecret) {
		t.Fatalf("RedactDSN() = %q, want password query redacted", got)
	}
}

func TestConfigRedactedHidesRedisPassword(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Database.DSN = "postgres://gombit:db-secret@localhost:5432/app" // #nosec G101 -- fake local test DSN.
	cfg.Cache.Redis.Password = "redis-secret"
	cfg.Auth.JWTSecret = "jwt-super-secret"

	got := cfg.Redacted()
	if strings.Contains(got.Database.DSN, "db-secret") {
		t.Fatalf("Redacted DSN = %q, still contains db password", got.Database.DSN)
	}
	if got.Cache.Redis.Password != RedactedSecret {
		t.Fatalf("Redacted Redis password = %q, want %s", got.Cache.Redis.Password, RedactedSecret)
	}
	if got.Auth.JWTSecret != RedactedSecret {
		t.Fatalf("Redacted JWT secret = %q, want %s", got.Auth.JWTSecret, RedactedSecret)
	}
	if cfg.Cache.Redis.Password != "redis-secret" {
		t.Fatal("Redacted() mutated the original Redis password")
	}
	if cfg.Auth.JWTSecret != "jwt-super-secret" {
		t.Fatal("Redacted() mutated the original JWT secret")
	}
}

func TestSanitizeErrorRemovesSecrets(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Database.DSN = "postgres://gombit:db-secret@localhost:5432/app" // #nosec G101 -- fake local test DSN.
	cfg.Cache.Redis.Password = "redis-secret"
	cfg.Auth.JWTSecret = "jwt-super-secret"

	err := errors.New("dial postgres://gombit:db-secret@localhost:5432/app with redis-secret jwt-super-secret")
	got := SanitizeError(err, cfg)
	if strings.Contains(got, "db-secret") || strings.Contains(got, "redis-secret") || strings.Contains(got, "jwt-super-secret") {
		t.Fatalf("SanitizeError() = %q, still contains secrets", got)
	}
}
