package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	envAppName   = "GOMBIT_APP_NAME"
	envEnv       = "GOMBIT_ENV"
	envHTTPAddr  = "GOMBIT_HTTP_ADDR"
	envAPIPrefix = "GOMBIT_API_PREFIX"
)

// Environment is the runtime environment name.
type Environment string

const (
	// EnvironmentDevelopment is the default local development environment.
	EnvironmentDevelopment Environment = "development"
	// EnvironmentTest is used by tests and ephemeral verification.
	EnvironmentTest Environment = "test"
	// EnvironmentProduction is used by deployed production applications.
	EnvironmentProduction Environment = "production"
)

// Config is the typed configuration consumed by Gombit runtime packages.
type Config struct {
	AppName     string
	Environment Environment
	HTTP        HTTPConfig
	API         APIConfig
}

// HTTPConfig contains HTTP server configuration.
type HTTPConfig struct {
	Addr string
}

// APIConfig contains public API configuration.
type APIConfig struct {
	Prefix string
}

// EnvLookup reads an environment variable by name.
type EnvLookup func(key string) (value string, ok bool)

// Default returns the default development configuration.
func Default() Config {
	return Config{
		AppName:     "Gombit",
		Environment: EnvironmentDevelopment,
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		API: APIConfig{
			Prefix: "/api/v1",
		},
	}
}

// Load reads configuration from the process environment and validates it.
func Load() (Config, error) {
	return LoadFromEnv(os.LookupEnv)
}

// LoadFromEnv reads configuration through lookup and validates it.
func LoadFromEnv(lookup EnvLookup) (Config, error) {
	if lookup == nil {
		return Config{}, errors.New("config: nil environment lookup")
	}

	cfg := Default()
	applyString(lookup, envAppName, &cfg.AppName)
	applyEnvironment(lookup, envEnv, &cfg.Environment)
	applyString(lookup, envHTTPAddr, &cfg.HTTP.Addr)
	applyString(lookup, envAPIPrefix, &cfg.API.Prefix)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate returns explicit field errors for invalid configuration values.
func (c Config) Validate() error {
	var errs FieldErrors

	if strings.TrimSpace(c.AppName) == "" {
		errs = append(errs, FieldError{
			Field:   "AppName",
			Env:     envAppName,
			Value:   c.AppName,
			Message: "must not be empty",
		})
	}

	switch c.Environment {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentProduction:
	default:
		errs = append(errs, FieldError{
			Field:   "Environment",
			Env:     envEnv,
			Value:   string(c.Environment),
			Message: "must be one of development, test, production",
		})
	}

	if strings.TrimSpace(c.HTTP.Addr) == "" {
		errs = append(errs, FieldError{
			Field:   "HTTP.Addr",
			Env:     envHTTPAddr,
			Value:   c.HTTP.Addr,
			Message: "must not be empty",
		})
	}

	if !strings.HasPrefix(c.API.Prefix, "/") {
		errs = append(errs, FieldError{
			Field:   "API.Prefix",
			Env:     envAPIPrefix,
			Value:   c.API.Prefix,
			Message: "must start with /",
		})
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func applyString(lookup EnvLookup, key string, dest *string) {
	if value, ok := lookup(key); ok {
		*dest = strings.TrimSpace(value)
	}
}

func applyEnvironment(lookup EnvLookup, key string, dest *Environment) {
	if value, ok := lookup(key); ok {
		*dest = Environment(strings.TrimSpace(value))
	}
}

// FieldError is one explicit configuration validation failure.
type FieldError struct {
	Field   string
	Env     string
	Value   string
	Message string
}

// Error returns a human-readable validation message.
func (e FieldError) Error() string {
	if e.Env == "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}

	return fmt.Sprintf("%s (%s): %s", e.Field, e.Env, e.Message)
}

// FieldErrors is a set of configuration validation failures.
type FieldErrors []FieldError

// Error returns a human-readable validation summary.
func (e FieldErrors) Error() string {
	switch len(e) {
	case 0:
		return "config: no validation errors"
	case 1:
		return "config: " + e[0].Error()
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "config: %d validation errors:", len(e))
		for _, err := range e {
			fmt.Fprintf(&b, " %s", err.Error())
		}
		return b.String()
	}
}
