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

## Environment

`config.Load()` reads the process environment and validates the resulting
configuration. The M1-1 boundary recognizes:

| Variable | Field | Default |
| --- | --- | --- |
| `GOMBIT_APP_NAME` | `Config.AppName` | `Gombit` |
| `GOMBIT_ENV` | `Config.Environment` | `development` |
| `GOMBIT_HTTP_ADDR` | `Config.HTTP.Addr` | `:8080` |
| `GOMBIT_API_PREFIX` | `Config.API.Prefix` | `/api/v1` |

`GOMBIT_ENV` accepts the exact lowercase values `development`, `test`, and
`production`.

Validation returns `config.FieldErrors`, which names the typed field, the
environment variable, the invalid value, and the validation message.

Appendix C production checks, such as JWT secret strength, secure cookies,
CORS credentials, debug Gin mode, database settings, and Redis settings, land
with the features that introduce those typed fields. This boundary only
validates the keys listed above. Future secret-bearing fields must not copy
secret values into `FieldError.Value`.

Runtime extraction work should accept `config.Config` values instead of calling
`os.Getenv` or `os.LookupEnv` directly.
