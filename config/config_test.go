package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
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
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestLoadFromEnv(t *testing.T) {
	env := map[string]string{
		envAppName:   " Example ",
		envEnv:       " test ",
		envHTTPAddr:  " 127.0.0.1:9000 ",
		envAPIPrefix: " /api ",
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
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadFromEnv() = %#v, want %#v", got, want)
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
	}
	if !reflect.DeepEqual([]FieldError(fieldErrors), want) {
		t.Fatalf("Validate() field errors = %#v, want %#v", []FieldError(fieldErrors), want)
	}

	message := err.Error()
	for _, wantPart := range []string{envAppName, envEnv, envHTTPAddr, envAPIPrefix} {
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

func mapLookup(values map[string]string) EnvLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
