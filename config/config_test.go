package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	got := Default()

	if got.AppName != "Gombit" {
		t.Fatalf("AppName = %q, want %q", got.AppName, "Gombit")
	}
	if got.Environment != EnvironmentDevelopment {
		t.Fatalf("Environment = %q, want %q", got.Environment, EnvironmentDevelopment)
	}
	if got.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want %q", got.HTTP.Addr, ":8080")
	}
	if got.API.Prefix != "/api/v1" {
		t.Fatalf("API.Prefix = %q, want %q", got.API.Prefix, "/api/v1")
	}
	if got.Database.Driver != DatabaseDriverSQLite {
		t.Fatalf("Database.Driver = %q, want %q", got.Database.Driver, DatabaseDriverSQLite)
	}
	if got.Database.DSN != "file:gombit.db?cache=shared&_fk=1" {
		t.Fatalf("Database.DSN = %q, want default sqlite DSN", got.Database.DSN)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestLoadFromEnv(t *testing.T) {
	env := map[string]string{
		envAppName:                 " Example ",
		envEnv:                     " test ",
		envHTTPAddr:                " 127.0.0.1:9000 ",
		envAPIPrefix:               " /api ",
		envDatabaseDriver:          " postgres ",
		envDatabaseDSN:             " host=localhost user=gombit dbname=app sslmode=disable ",
		envDatabaseMaxOpenConns:    " 20 ",
		envDatabaseMaxIdleConns:    " 4 ",
		envDatabaseConnMaxLifetime: " 45m ",
	}

	got, err := LoadFromEnv(mapLookup(env))
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v, want nil", err)
	}

	want := Config{
		AppName:     "Example",
		Environment: EnvironmentTest,
		HTTP: HTTPConfig{
			Addr: "127.0.0.1:9000",
		},
		API: APIConfig{
			Prefix: "/api",
		},
		Database: DatabaseConfig{
			Driver:          DatabaseDriverPostgres,
			DSN:             "host=localhost user=gombit dbname=app sslmode=disable",
			MaxOpenConns:    20,
			MaxIdleConns:    4,
			ConnMaxLifetime: 45 * time.Minute,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadFromEnv() = %#v, want %#v", got, want)
	}
}

func TestLoadFromEnvUsesDefaultsWhenUnset(t *testing.T) {
	got, err := LoadFromEnv(mapLookup(nil))
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("LoadFromEnv() = %#v, want default %#v", got, Default())
	}
}

func TestLoadUsesProcessEnvironment(t *testing.T) {
	t.Setenv(envAppName, "Process Example")
	t.Setenv(envEnv, string(EnvironmentProduction))
	t.Setenv(envHTTPAddr, ":9090")
	t.Setenv(envAPIPrefix, "/api")
	t.Setenv(envDatabaseDriver, string(DatabaseDriverMySQL))
	t.Setenv(envDatabaseDSN, "gombit@tcp(localhost:3306)/app?parseTime=true")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	want := Config{
		AppName:     "Process Example",
		Environment: EnvironmentProduction,
		HTTP: HTTPConfig{
			Addr: ":9090",
		},
		API: APIConfig{
			Prefix: "/api",
		},
		Database: DatabaseConfig{
			Driver: DatabaseDriverMySQL,
			DSN:    "gombit@tcp(localhost:3306)/app?parseTime=true",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadFromEnvRequiresLookup(t *testing.T) {
	_, err := LoadFromEnv(nil)
	if err == nil {
		t.Fatal("LoadFromEnv(nil) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "nil environment lookup") {
		t.Fatalf("LoadFromEnv(nil) error = %q, want nil lookup message", err)
	}
}

func TestValidateReportsExplicitFieldErrors(t *testing.T) {
	cfg := Config{
		AppName:     " ",
		Environment: "staging",
		HTTP: HTTPConfig{
			Addr: "",
		},
		API: APIConfig{
			Prefix: "api",
		},
		Database: DatabaseConfig{
			Driver:          "oracle",
			DSN:             "",
			MaxOpenConns:    -1,
			MaxIdleConns:    2,
			ConnMaxLifetime: -time.Second,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation errors")
	}

	var fieldErrors FieldErrors
	if !errors.As(err, &fieldErrors) {
		t.Fatalf("Validate() error type = %T, want FieldErrors", err)
	}

	want := []FieldError{
		{Field: "AppName", Env: envAppName, Value: " ", Message: "must not be empty"},
		{
			Field:   "Environment",
			Env:     envEnv,
			Value:   "staging",
			Message: "must be one of development, test, production",
		},
		{Field: "HTTP.Addr", Env: envHTTPAddr, Value: "", Message: "must not be empty"},
		{Field: "API.Prefix", Env: envAPIPrefix, Value: "api", Message: "must start with /"},
		{
			Field:   "Database.Driver",
			Env:     envDatabaseDriver,
			Value:   "oracle",
			Message: "must be one of sqlite, postgres, mysql",
		},
		{Field: "Database.DSN", Env: envDatabaseDSN, Message: "must not be empty"},
		{
			Field:   "Database.MaxOpenConns",
			Env:     envDatabaseMaxOpenConns,
			Value:   "-1",
			Message: "must be greater than or equal to zero",
		},
		{
			Field:   "Database.MaxIdleConns",
			Env:     envDatabaseMaxIdleConns,
			Value:   "2",
			Message: "must be less than or equal to Database.MaxOpenConns",
		},
		{
			Field:   "Database.ConnMaxLifetime",
			Env:     envDatabaseConnMaxLifetime,
			Value:   "-1s",
			Message: "must be greater than or equal to zero",
		},
	}
	if !reflect.DeepEqual([]FieldError(fieldErrors), want) {
		t.Fatalf("Validate() field errors = %#v, want %#v", []FieldError(fieldErrors), want)
	}

	message := err.Error()
	for _, wantPart := range []string{
		envAppName,
		envEnv,
		envHTTPAddr,
		envAPIPrefix,
		envDatabaseDriver,
		envDatabaseDSN,
		envDatabaseMaxOpenConns,
		envDatabaseMaxIdleConns,
		envDatabaseConnMaxLifetime,
	} {
		if !strings.Contains(message, wantPart) {
			t.Fatalf("Validate() error = %q, want it to contain %q", message, wantPart)
		}
	}
}

func TestLoadFromEnvReturnsValidationErrors(t *testing.T) {
	_, err := LoadFromEnv(mapLookup(map[string]string{
		envEnv:       "prod",
		envAPIPrefix: "api",
	}))
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want validation errors")
	}

	var fieldErrors FieldErrors
	if !errors.As(err, &fieldErrors) {
		t.Fatalf("LoadFromEnv() error type = %T, want FieldErrors", err)
	}
	if len(fieldErrors) != 2 {
		t.Fatalf("LoadFromEnv() field error count = %d, want 2", len(fieldErrors))
	}
}

func TestLoadFromEnvReturnsParseErrors(t *testing.T) {
	_, err := LoadFromEnv(mapLookup(map[string]string{
		envDatabaseMaxOpenConns:    "many",
		envDatabaseConnMaxLifetime: "later",
	}))
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want parse errors")
	}

	var fieldErrors FieldErrors
	if !errors.As(err, &fieldErrors) {
		t.Fatalf("LoadFromEnv() error type = %T, want FieldErrors", err)
	}
	if len(fieldErrors) != 2 {
		t.Fatalf("LoadFromEnv() field error count = %d, want 2", len(fieldErrors))
	}
}

func mapLookup(values map[string]string) EnvLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
