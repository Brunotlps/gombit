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
	envHTTPTrustedProxies      = "GOMBIT_HTTP_TRUSTED_PROXIES"
	envHTTPRequestTimeout      = "GOMBIT_HTTP_REQUEST_TIMEOUT"
	envAPIPrefix               = "GOMBIT_API_PREFIX"
	envDocsEnabled             = "GOMBIT_DOCS_ENABLED"
	envDatabaseDriver          = "GOMBIT_DATABASE_DRIVER"
	envDatabaseDSN             = "GOMBIT_DATABASE_DSN"
	envDatabaseMaxOpenConns    = "GOMBIT_DATABASE_MAX_OPEN_CONNS"
	envDatabaseMaxIdleConns    = "GOMBIT_DATABASE_MAX_IDLE_CONNS"
	envDatabaseConnMaxLifetime = "GOMBIT_DATABASE_CONN_MAX_LIFETIME"
	envCacheDriver             = "GOMBIT_CACHE_DRIVER"
	envCacheNamespace          = "GOMBIT_CACHE_NAMESPACE"
	envRedisAddr               = "GOMBIT_REDIS_ADDR"
	envRedisUsername           = "GOMBIT_REDIS_USERNAME"
	envRedisPassword           = "GOMBIT_REDIS_PASSWORD" // #nosec G101 -- environment variable name, not a credential.
	envRedisDB                 = "GOMBIT_REDIS_DB"
	envRedisDialTimeout        = "GOMBIT_REDIS_DIAL_TIMEOUT"
	envRedisReadTimeout        = "GOMBIT_REDIS_READ_TIMEOUT"
	envRedisWriteTimeout       = "GOMBIT_REDIS_WRITE_TIMEOUT"
	envRedisTLS                = "GOMBIT_REDIS_TLS"
	envRedisTLSInsecure        = "GOMBIT_REDIS_TLS_INSECURE"
	envLogLevel                = "GOMBIT_LOG_LEVEL"
	envLogSink                 = "GOMBIT_LOG_SINK"
	envJWTSecret               = "GOMBIT_JWT_SECRET" // #nosec G101 -- environment variable name, not a credential.
	envJWTAccessTTL            = "GOMBIT_JWT_ACCESS_TTL"
	envJWTRefreshTTL           = "GOMBIT_JWT_REFRESH_TTL"
)

const (
	// MinProductionJWTSecretLength is the Appendix C floor for production
	// HMAC secrets. Development and test may use a shorter or empty secret
	// (empty disables Bearer auth).
	MinProductionJWTSecretLength = 32

	// DefaultAccessTokenTTL is the signed access-token lifetime.
	DefaultAccessTokenTTL = 15 * time.Minute

	// DefaultRefreshTokenTTL is the refresh-token lifetime. Refresh tokens
	// rotate on each successful refresh.
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour
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
	Cache       CacheConfig
	Logging     LoggingConfig
	Auth        AuthConfig
}

// HTTPConfig contains HTTP server configuration.
type HTTPConfig struct {
	Addr           string
	TrustedProxies []string
	RequestTimeout time.Duration
}

// APIConfig contains public API configuration.
type APIConfig struct {
	Prefix string
	// DocsEnabled serves the FastAPI-style UI at /docs.
	// Default() (development) leaves this true. Changing Environment on a
	// Default() value does not flip it — use DefaultFor(env) or set this
	// field. LoadFromEnv uses DefaultDocsEnabled when GOMBIT_DOCS_ENABLED
	// is unset.
	DocsEnabled bool
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

// CacheDriver names a supported cache driver.
type CacheDriver string

const (
	// CacheDriverMemory stores cache values in process memory.
	CacheDriverMemory CacheDriver = "memory"
	// CacheDriverRedis stores cache values in Redis.
	CacheDriverRedis CacheDriver = "redis"
	// CacheDriverNoop disables persistence while preserving cache call sites.
	CacheDriverNoop CacheDriver = "noop"
)

// CacheConfig contains cache configuration consumed by cache.Open.
type CacheConfig struct {
	Driver    CacheDriver
	Namespace string
	Redis     RedisConfig
}

// RedisConfig contains go-redis client configuration.
type RedisConfig struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TLS          bool
	TLSInsecure  bool
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

// AuthConfig contains Bearer JWT configuration. Cookie/session mode is M5-3.
type AuthConfig struct {
	// JWTSecret is the HMAC key for access tokens. It is never a VITE_* value.
	// Empty disables framework-owned auth routes outside production.
	JWTSecret string
	// AccessTokenTTL is the signed access-token lifetime.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is the refresh-token lifetime.
	RefreshTokenTTL time.Duration
	// BcryptCost is the bcrypt cost for password hashes. Zero means
	// bcrypt.DefaultCost. Tests may set a lower cost.
	BcryptCost int
}

// Enabled reports whether Bearer auth should be mounted (JWT secret is set).
func (c AuthConfig) Enabled() bool {
	return strings.TrimSpace(c.JWTSecret) != ""
}

// EnvLookup reads an environment variable by name.
type EnvLookup func(key string) (value string, ok bool)

// Default returns the default development configuration, including /docs on.
// It does not follow Environment: mutating Environment on the result leaves
// DocsEnabled and the cache namespace at development values. Use DefaultFor
// when constructing a non-development config for framework.WithConfig.
func Default() Config {
	return DefaultFor(EnvironmentDevelopment)
}

// DefaultFor returns Default-shaped configuration for env, including
// environment-derived DocsEnabled and cache namespace.
func DefaultFor(env Environment) Config {
	if env == "" {
		env = EnvironmentDevelopment
	}
	return Config{
		AppName:     "Gombit",
		Environment: env,
		HTTP: HTTPConfig{
			Addr:           ":8080",
			RequestTimeout: 60 * time.Second,
		},
		API: APIConfig{
			Prefix:      "/api/v1",
			DocsEnabled: DefaultDocsEnabled(env),
		},
		Database: DatabaseConfig{
			Driver: DatabaseDriverSQLite,
			DSN:    "file:gombit.db?cache=shared&_fk=1",
		},
		Cache: CacheConfig{
			Driver:    CacheDriverMemory,
			Namespace: DefaultCacheNamespace("Gombit", env),
			Redis: RedisConfig{
				Addr:         "127.0.0.1:6379",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
		Logging: LoggingConfig{
			Level: LogLevelInfo,
			Sink:  LogSinkStderr,
		},
		Auth: defaultAuthConfig(),
	}
}

func defaultAuthConfig() AuthConfig {
	return AuthConfig{
		AccessTokenTTL:  DefaultAccessTokenTTL,
		RefreshTokenTTL: DefaultRefreshTokenTTL,
	}
}

// DefaultDocsEnabled reports whether /docs is on when GOMBIT_DOCS_ENABLED is
// unset. Production is off; development and test are on.
func DefaultDocsEnabled(env Environment) bool {
	return env != EnvironmentProduction
}

// ApplyEnvironmentDefaults sets fields that follow Environment when the
// caller did not set them explicitly. LoadFromEnv uses this when
// GOMBIT_DOCS_ENABLED is unset. WithConfig callers that only change
// Environment on Default() should use DefaultFor instead.
func ApplyEnvironmentDefaults(cfg Config) Config {
	cfg.API.DocsEnabled = DefaultDocsEnabled(cfg.Environment)
	return cfg
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
	applyStringList(lookup, envHTTPTrustedProxies, &cfg.HTTP.TrustedProxies)
	applyString(lookup, envAPIPrefix, &cfg.API.Prefix)
	_, docsEnabledSet := lookup(envDocsEnabled)

	var errs FieldErrors
	applyDuration(
		lookup,
		envHTTPRequestTimeout,
		"HTTP.RequestTimeout",
		&cfg.HTTP.RequestTimeout,
		&errs,
	)
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
	applyCacheDriver(lookup, envCacheDriver, &cfg.Cache.Driver)
	_, cacheNamespaceSet := lookup(envCacheNamespace)
	applyString(lookup, envCacheNamespace, &cfg.Cache.Namespace)
	applyString(lookup, envRedisAddr, &cfg.Cache.Redis.Addr)
	applyString(lookup, envRedisUsername, &cfg.Cache.Redis.Username)
	applyString(lookup, envRedisPassword, &cfg.Cache.Redis.Password)
	applyInt(lookup, envRedisDB, "Cache.Redis.DB", &cfg.Cache.Redis.DB, &errs)
	applyDuration(lookup, envRedisDialTimeout, "Cache.Redis.DialTimeout", &cfg.Cache.Redis.DialTimeout, &errs)
	applyDuration(lookup, envRedisReadTimeout, "Cache.Redis.ReadTimeout", &cfg.Cache.Redis.ReadTimeout, &errs)
	applyDuration(lookup, envRedisWriteTimeout, "Cache.Redis.WriteTimeout", &cfg.Cache.Redis.WriteTimeout, &errs)
	applyBool(lookup, envRedisTLS, "Cache.Redis.TLS", &cfg.Cache.Redis.TLS, &errs)
	applyBool(lookup, envRedisTLSInsecure, "Cache.Redis.TLSInsecure", &cfg.Cache.Redis.TLSInsecure, &errs)
	if !cacheNamespaceSet {
		cfg.Cache.Namespace = DefaultCacheNamespace(cfg.AppName, cfg.Environment)
	}
	applyLogLevel(lookup, envLogLevel, &cfg.Logging.Level)
	applyLogSink(lookup, envLogSink, &cfg.Logging.Sink)
	applyString(lookup, envJWTSecret, &cfg.Auth.JWTSecret)
	applyDuration(lookup, envJWTAccessTTL, "Auth.AccessTokenTTL", &cfg.Auth.AccessTokenTTL, &errs)
	applyDuration(lookup, envJWTRefreshTTL, "Auth.RefreshTokenTTL", &cfg.Auth.RefreshTokenTTL, &errs)
	if docsEnabledSet {
		applyBool(lookup, envDocsEnabled, "API.DocsEnabled", &cfg.API.DocsEnabled, &errs)
	} else {
		cfg = ApplyEnvironmentDefaults(cfg)
	}

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

	if c.HTTP.RequestTimeout < 0 {
		errs = append(errs, FieldError{
			Field:   "HTTP.RequestTimeout",
			Env:     envHTTPRequestTimeout,
			Value:   c.HTTP.RequestTimeout.String(),
			Message: "must be greater than or equal to zero",
		})
	}

	for _, proxy := range c.HTTP.TrustedProxies {
		if strings.TrimSpace(proxy) == "" {
			errs = append(errs, FieldError{
				Field:   "HTTP.TrustedProxies",
				Env:     envHTTPTrustedProxies,
				Value:   proxy,
				Message: "must not contain empty entries",
			})
		}
		if c.Environment == EnvironmentProduction && isUnsafeTrustedProxy(proxy) {
			errs = append(errs, FieldError{
				Field:   "HTTP.TrustedProxies",
				Env:     envHTTPTrustedProxies,
				Value:   proxy,
				Message: "must not trust all proxies in production",
			})
		}
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
	validateCacheConfig(&errs, c.Environment, c.Cache)
	validateLoggingConfig(&errs, c.Logging)
	validateAuthConfig(&errs, c.Environment, c.Auth)

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

// ValidateCache returns explicit field errors for invalid cache settings.
func ValidateCache(cfg CacheConfig) error {
	var errs FieldErrors
	validateCacheConfig(&errs, "", cfg)
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

// DefaultCacheNamespace returns the conventional cache namespace for app/env.
func DefaultCacheNamespace(appName string, env Environment) string {
	name := strings.ToLower(strings.TrimSpace(appName))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	normalized := strings.Trim(b.String(), "-")
	if normalized == "" {
		normalized = "gombit"
	}
	return normalized + ":" + string(env)
}

func validateAuthConfig(errs *FieldErrors, env Environment, cfg AuthConfig) {
	if cfg.AccessTokenTTL < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Auth.AccessTokenTTL",
			Env:     envJWTAccessTTL,
			Value:   cfg.AccessTokenTTL.String(),
			Message: "must be greater than or equal to zero",
		})
	}
	if cfg.RefreshTokenTTL < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Auth.RefreshTokenTTL",
			Env:     envJWTRefreshTTL,
			Value:   cfg.RefreshTokenTTL.String(),
			Message: "must be greater than or equal to zero",
		})
	}
	if cfg.BcryptCost < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Auth.BcryptCost",
			Message: "must be greater than or equal to zero",
		})
	}
	if env == EnvironmentProduction && cfg.JWTSecret != "" && len(cfg.JWTSecret) < MinProductionJWTSecretLength {
		*errs = append(*errs, FieldError{
			Field: "Auth.JWTSecret",
			Env:   envJWTSecret,
			Message: fmt.Sprintf(
				"must be at least %d characters in production",
				MinProductionJWTSecretLength,
			),
		})
	}
	if cfg.Enabled() {
		if cfg.AccessTokenTTL <= 0 {
			*errs = append(*errs, FieldError{
				Field:   "Auth.AccessTokenTTL",
				Env:     envJWTAccessTTL,
				Value:   cfg.AccessTokenTTL.String(),
				Message: "must be greater than zero when a JWT secret is set",
			})
		}
		if cfg.RefreshTokenTTL <= 0 {
			*errs = append(*errs, FieldError{
				Field:   "Auth.RefreshTokenTTL",
				Env:     envJWTRefreshTTL,
				Value:   cfg.RefreshTokenTTL.String(),
				Message: "must be greater than zero when a JWT secret is set",
			})
		}
	}
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

func validateCacheConfig(errs *FieldErrors, env Environment, cfg CacheConfig) {
	validateRedis := false
	switch cfg.Driver {
	case CacheDriverMemory, CacheDriverNoop:
	case CacheDriverRedis:
		validateRedis = true
	default:
		validateRedis = true
		*errs = append(*errs, FieldError{
			Field:   "Cache.Driver",
			Env:     envCacheDriver,
			Value:   string(cfg.Driver),
			Message: "must be one of memory, redis, noop",
		})
	}

	if strings.TrimSpace(cfg.Namespace) == "" {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Namespace",
			Env:     envCacheNamespace,
			Value:   cfg.Namespace,
			Message: "must not be empty",
		})
	}

	if !validateRedis {
		return
	}

	if strings.TrimSpace(cfg.Redis.Addr) == "" {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Redis.Addr",
			Env:     envRedisAddr,
			Value:   cfg.Redis.Addr,
			Message: "must not be empty",
		})
	}

	if cfg.Redis.DB < 0 {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Redis.DB",
			Env:     envRedisDB,
			Value:   strconv.Itoa(cfg.Redis.DB),
			Message: "must be greater than or equal to zero",
		})
	}

	if cfg.Redis.DialTimeout <= 0 {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Redis.DialTimeout",
			Env:     envRedisDialTimeout,
			Value:   cfg.Redis.DialTimeout.String(),
			Message: "must be greater than zero",
		})
	}
	if cfg.Redis.ReadTimeout <= 0 {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Redis.ReadTimeout",
			Env:     envRedisReadTimeout,
			Value:   cfg.Redis.ReadTimeout.String(),
			Message: "must be greater than zero",
		})
	}
	if cfg.Redis.WriteTimeout <= 0 {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Redis.WriteTimeout",
			Env:     envRedisWriteTimeout,
			Value:   cfg.Redis.WriteTimeout.String(),
			Message: "must be greater than zero",
		})
	}

	if env == EnvironmentProduction && cfg.Redis.TLSInsecure {
		*errs = append(*errs, FieldError{
			Field:   "Cache.Redis.TLSInsecure",
			Env:     envRedisTLSInsecure,
			Value:   strconv.FormatBool(cfg.Redis.TLSInsecure),
			Message: "must be false in production",
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

func applyStringList(lookup EnvLookup, key string, dest *[]string) {
	value, ok := lookup(key)
	if !ok {
		return
	}

	var parsed []string
	for _, part := range strings.Split(value, ",") {
		if item := strings.TrimSpace(part); item != "" {
			parsed = append(parsed, item)
		}
	}
	*dest = parsed
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

func applyCacheDriver(lookup EnvLookup, key string, dest *CacheDriver) {
	if value, ok := lookup(key); ok {
		*dest = CacheDriver(strings.TrimSpace(value))
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

func applyBool(lookup EnvLookup, key string, field string, dest *bool, errs *FieldErrors) {
	value, ok := lookup(key)
	if !ok {
		return
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		*dest = true
	case "0", "false", "no", "off":
		*dest = false
	default:
		*errs = append(*errs, FieldError{
			Field:   field,
			Env:     key,
			Value:   value,
			Message: "must be a boolean",
		})
	}
}

func isUnsafeTrustedProxy(proxy string) bool {
	switch strings.TrimSpace(proxy) {
	case "*", "0.0.0.0/0", "::/0":
		return true
	default:
		return false
	}
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
