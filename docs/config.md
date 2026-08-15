# Typed Configuration

Gombit's runtime packages receive typed configuration through
`config.Config`. Environment reads belong at this boundary, not in low-level
runtime packages.

## Defaults

`config.Default()` returns the current development defaults:

- app name: `Gombit`
- environment: `development`
- HTTP address: `:8080`
- API prefix: `/api/v1`
- database driver: `sqlite`
- database DSN: `file:gombit.db?cache=shared&_fk=1`
- log level: `info`
- log sink: `stderr`

## Environment

`config.Load()` reads the process environment and validates the resulting
configuration. The M1-1 boundary recognizes:

| Variable | Field | Default |
| --- | --- | --- |
| `GOMBIT_APP_NAME` | `Config.AppName` | `Gombit` |
| `GOMBIT_ENV` | `Config.Environment` | `development` |
| `GOMBIT_HTTP_ADDR` | `Config.HTTP.Addr` | `:8080` |
| `GOMBIT_API_PREFIX` | `Config.API.Prefix` | `/api/v1` |
| `GOMBIT_DATABASE_DRIVER` | `Config.Database.Driver` | `sqlite` |
| `GOMBIT_DATABASE_DSN` | `Config.Database.DSN` | `file:gombit.db?cache=shared&_fk=1` |
| `GOMBIT_DATABASE_MAX_OPEN_CONNS` | `Config.Database.MaxOpenConns` | `0` |
| `GOMBIT_DATABASE_MAX_IDLE_CONNS` | `Config.Database.MaxIdleConns` | `0` |
| `GOMBIT_DATABASE_CONN_MAX_LIFETIME` | `Config.Database.ConnMaxLifetime` | `0` |
| `GOMBIT_LOG_LEVEL` | `Config.Logging.Level` | `info` |
| `GOMBIT_LOG_SINK` | `Config.Logging.Sink` | `stderr` |

`GOMBIT_ENV` accepts the exact lowercase values `development`, `test`, and
`production`.
`GOMBIT_DATABASE_DRIVER` accepts `sqlite`, `postgres`, and `mysql`.
`GOMBIT_DATABASE_CONN_MAX_LIFETIME` uses Go duration syntax such as `30m` or
`1h`.
`GOMBIT_LOG_LEVEL` accepts `debug`, `info`, `warn`, and `error`.
`GOMBIT_LOG_SINK` accepts `stderr`, `stdout`, and `mongo`; Mongo logging is an
external module hook, not a runtime dependency.

Validation returns `config.FieldErrors`, which names the typed field, the
environment variable, the invalid value, and the validation message.

Appendix C production checks, such as JWT secret strength, secure cookies,
CORS credentials, debug Gin mode, and Redis settings land with the features
that introduce those typed fields. Future secret-bearing fields must not copy
secret values into `FieldError.Value`; `Config.Database.DSN` validation does
not echo the DSN value.

Runtime extraction work should accept `config.Config` values instead of calling
`os.Getenv` or `os.LookupEnv` directly.
