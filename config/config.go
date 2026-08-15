package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envAppName                 = "GOMBIT_APP_NAME"
	envEnv                     = "GOMBIT_ENV"
	envHTTPAddr                = "GOMBIT_HTTP_ADDR"
	envAPIPrefix               = "GOMBIT_API_PREFIX"
	envDatabaseDriver          = "GOMBIT_DATABASE_DRIVER"
	envDatabaseDSN             = "GOMBIT_DATABASE_DSN"
	envDatabaseMaxOpenConns    = "GOMBIT_DATABASE_MAX_OPEN_CONNS"
	envDatabaseMaxIdleConns    = "GOMBIT_DATABASE_MAX_IDLE_CONNS"
	envDatabaseConnMaxLifetime = "GOMBIT_DATABASE_CONN_MAX_LIFETIME"
	envLogLevel                = "GOMBIT_LOG_LEVEL"
	envLogSink                 = "GOMBIT_LOG_SINK"
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
	Database    DatabaseConfig
	Logging     LoggingConfig
}

// HTTPConfig contains HTTP server configuration.
type HTTPConfig struct {
	Addr string
}

// APIConfig contains public API configuration.
type APIConfig struct {
	Prefix string
}

// DatabaseDriver names a supported SQL database driver.
type DatabaseDriver string

const (
	// DatabaseDriverSQLite selects the GORM SQLite dialector.
	DatabaseDriverSQLite DatabaseDriver = "sqlite"
	// DatabaseDriverPostgres selects the GORM PostgreSQL dialector.
	DatabaseDriverPostgres DatabaseDriver = "postgres"
	// DatabaseDriverMySQL selects the GORM MySQL dialector.
	DatabaseDriverMySQL DatabaseDriver = "mysql"
)

// DatabaseConfig contains SQL database configuration consumed by database.Open.
type DatabaseConfig struct {
	Driver          DatabaseDriver
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// LogLevel names the configured logging level.
type LogLevel string

const (
	// LogLevelDebug enables debug, info, warn, and error logs.
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo enables info, warn, and error logs.
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn enables warn and error logs.
	LogLevelWarn LogLevel = "warn"
	// LogLevelError enables error logs.
	LogLevelError LogLevel = "error"
)

// LogSink names the configured logging sink.
type LogSink string

const (
	// LogSinkStderr writes JSON logs to stderr.
	LogSinkStderr LogSink = "stderr"
	// LogSinkStdout writes JSON logs to stdout.
	LogSinkStdout LogSink = "stdout"
	// LogSinkMongo selects an external Mongo logging module.
	LogSinkMongo LogSink = "mongo"
)

// LoggingConfig contains structured logging configuration.
type LoggingConfig struct {
	Level LogLevel
	Sink  LogSink
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
		Database: DatabaseConfig{
			Driver: DatabaseDriverSQLite,
			DSN:    "file:gombit.db?cache=shared&_fk=1",
		},
		Logging: LoggingConfig{
			Level: LogLevelInfo,
			Sink:  LogSinkStderr,
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

	var errs FieldErrors
	applyDatabaseDriver(lookup, envDatabaseDriver, &cfg.Database.Driver)
	applyString(lookup, envDatabaseDSN, &cfg.Database.DSN)
	applyInt(lookup, envDatabaseMaxOpenConns, "Database.MaxOpenConns", &cfg.Database.MaxOpenConns, &errs)
	applyInt(lookup, envDatabaseMaxIdleConns, "Database.MaxIdleConns", &cfg.Database.MaxIdleConns, &errs)
	applyDuration(
		lookup,
		envDatabaseConnMaxLifetime,
		"Database.ConnMaxLifetime",
		&cfg.Database.ConnMaxLifetime,
		&errs,
	)
	applyLogLevel(lookup, envLogLevel, &cfg.Logging.Level)
	applyLogSink(lookup, envLogSink, &cfg.Logging.Sink)

	if err := cfg.Validate(); err != nil {
		var fieldErrors FieldErrors
		if !errors.As(err, &fieldErrors) {
			return Config{}, err
		}
		errs = append(errs, fieldErrors...)
	}
	if len(errs) > 0 {
		return Config{}, errs
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

	validateDatabaseConfig(&errs, c.Database)
	validateLoggingConfig(&errs, c.Logging)

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// ValidateDatabase returns explicit field errors for invalid database settings.
func ValidateDatabase(cfg DatabaseConfig) error {
	var errs FieldErrors
	validateDatabaseConfig(&errs, cfg)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateLogging returns explicit field errors for invalid logging settings.
func ValidateLogging(cfg LoggingConfig) error {
	var errs FieldErrors
	validateLoggingConfig(&errs, cfg)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateLoggingConfig(errs *FieldErrors, cfg LoggingConfig) {
	switch cfg.Level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		*errs = append(*errs, FieldError{
			Field:   "Logging.Level",
			Env:     envLogLevel,
			Value:   string(cfg.Level),
			Message: "must be one of debug, info, warn, error",
		})
	}

	switch cfg.Sink {
	case LogSinkStderr, LogSinkStdout, LogSinkMongo:
	default:
		*errs = append(*errs, FieldError{
			Field:   "Logging.Sink",
			Env:     envLogSink,
			Value:   string(cfg.Sink),
			Message: "must be one of stderr, stdout, mongo",
		})
	}
}

func validateDatabaseConfig(errs *FieldErrors, cfg DatabaseConfig) {
	switch cfg.Driver {
	case DatabaseDriverSQLite, DatabaseDriverPostgres, DatabaseDriverMySQL:
	default:
		*errs = append(*errs, FieldError{
			Field:   "Database.Driver",
			Env:     envDatabaseDriver,
			Value:   string(cfg.Driver),
			Message: "must be one of sqlite, postgres, mysql",
		})
	}

	if strings.TrimSpace(cfg.DSN) == "" {
		*errs = append(*errs, FieldError{
			Field:   "Database.DSN",
			Env:     envDatabaseDSN,
			Message: "must not be empty",
		})
	}

	if cfg.MaxOpenConns < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Database.MaxOpenConns",
			Env:     envDatabaseMaxOpenConns,
			Value:   strconv.Itoa(cfg.MaxOpenConns),
			Message: "must be greater than or equal to zero",
		})
	}

	if cfg.MaxIdleConns < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Database.MaxIdleConns",
			Env:     envDatabaseMaxIdleConns,
			Value:   strconv.Itoa(cfg.MaxIdleConns),
			Message: "must be greater than or equal to zero",
		})
	}

	if cfg.MaxOpenConns != 0 && cfg.MaxIdleConns > cfg.MaxOpenConns {
		*errs = append(*errs, FieldError{
			Field:   "Database.MaxIdleConns",
			Env:     envDatabaseMaxIdleConns,
			Value:   strconv.Itoa(cfg.MaxIdleConns),
			Message: "must be less than or equal to Database.MaxOpenConns",
		})
	}

	if cfg.ConnMaxLifetime < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Database.ConnMaxLifetime",
			Env:     envDatabaseConnMaxLifetime,
			Value:   cfg.ConnMaxLifetime.String(),
			Message: "must be greater than or equal to zero",
		})
	}
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

func applyDatabaseDriver(lookup EnvLookup, key string, dest *DatabaseDriver) {
	if value, ok := lookup(key); ok {
		*dest = DatabaseDriver(strings.TrimSpace(value))
	}
}

func applyLogLevel(lookup EnvLookup, key string, dest *LogLevel) {
	if value, ok := lookup(key); ok {
		*dest = LogLevel(strings.TrimSpace(value))
	}
}

func applyLogSink(lookup EnvLookup, key string, dest *LogSink) {
	if value, ok := lookup(key); ok {
		*dest = LogSink(strings.TrimSpace(value))
	}
}

func applyInt(lookup EnvLookup, key string, field string, dest *int, errs *FieldErrors) {
	value, ok := lookup(key)
	if !ok {
		return
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		*errs = append(*errs, FieldError{
			Field:   field,
			Env:     key,
			Value:   value,
			Message: "must be an integer",
		})
		return
	}
	*dest = parsed
}

func applyDuration(lookup EnvLookup, key string, field string, dest *time.Duration, errs *FieldErrors) {
	value, ok := lookup(key)
	if !ok {
		return
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		*errs = append(*errs, FieldError{
			Field:   field,
			Env:     key,
			Value:   value,
			Message: "must be a duration such as 30m or 1h",
		})
		return
	}
	*dest = parsed
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
