# Bearer JWT auth

Gombit's v0.1 API default is **Bearer JWT with refresh rotation** (C3 / D3).
Cookie/session + CSRF ([M5-3]) is a first-class alternative mode
(`gombit new --auth cookie`) — see [`docs/auth-cookie.md`](auth-cookie.md)
for its threat model; this page documents the Bearer default.
`gombit createsuperuser` ([M4-6]) is the CLI admin seed path; see
[cli.md](cli.md#gombit-createsuperuser).

Behavior lives in the `auth` runtime package. `framework.New` mounts the
Huma routes when `GOMBIT_JWT_SECRET` is set **and** a database is attached.
Generated apps (`gombit new --auth jwt`, the default) wire this through
`config.Load` + `framework.WithDatabase`; they do not copy handler code.

## Token storage (SPA)

| Token | Where | Notes |
| --- | --- | --- |
| Access JWT | **Memory only** | `Authorization: Bearer`. Never `localStorage` / `sessionStorage`. Lost on refresh, which is intended. |
| Refresh token | **Memory only** (JSON body) | Returned once on login/refresh. Rotated on each successful refresh. Not a cookie in v0.1. |

The refresh token stays in the JSON body (not a cookie) so this mode's
transport does not mix with `--auth cookie`'s HttpOnly session cookies
(see [`docs/auth-cookie.md`](auth-cookie.md)). Do not put tokens in
`VITE_*`.

## Endpoints

All paths use `config.API.Prefix` (default `/api/v1`). D10 envelopes.

| Method | Path | Auth |
| --- | --- | --- |
| `POST` | `/auth/register` | Public. Demo/bootstrap seed path (email + password); never sets `IsSuperuser`. |
| `POST` | `/auth/login` | Public. Returns `access_token`, `refresh_token`, `token_type`, `expires_in`. |
| `POST` | `/auth/refresh` | Public. Body `{ "refresh_token" }`. Issues a new pair; the old refresh token is invalid. |
| `POST` | `/auth/logout` | Public. Body `{ "refresh_token" }`. Revokes that refresh token. Bound access JWTs then fail. |
| `GET` | `/me` | Bearer access JWT. Example protected route for E2E. Missing/invalid Bearer is 401 with `WWW-Authenticate: Bearer realm="api"`. |

Passwords are hashed with bcrypt. Access JWTs are HS256, bound to the refresh
row so logout (and reuse of a rotated refresh token) 401s `/me`. Reusing a
revoked refresh token revokes that user's remaining refresh tokens.

`User.IsSuperuser` bypasses all permission checks.
`gombit createsuperuser` is the only built-in path that sets it;
`/auth/register` never does. Regular users receive `Permission` keys
directly or through `Group` membership. See [admin.md](admin.md) for the
ADMIN-3 assignment helpers and admin enforcement rules.

## Config

| Variable | Field | Default |
| --- | --- | --- |
| `GOMBIT_JWT_SECRET` | `Config.Auth.JWTSecret` | empty (auth unmounted) |
| `GOMBIT_JWT_ACCESS_TTL` | `Config.Auth.AccessTokenTTL` | `15m` |
| `GOMBIT_JWT_REFRESH_TTL` | `Config.Auth.RefreshTokenTTL` | `168h` (7 days) |

Production rejects a **non-empty** JWT secret shorter than 32 characters, and
the generated-app development placeholder, at `config.Load` / `Validate` and
`gombit doctor` (Appendix C). The secret is redacted by `Config.Redacted()`
and `gombit config show`. Empty in production leaves Bearer auth off; set a
long random secret to enable it.

`Config.Auth.Mode` (`GOMBIT_AUTH_MODE`, default `jwt`) selects between this
Bearer surface and `cookie` mode; see
[`docs/auth-cookie.md`](auth-cookie.md#config) for the cookie-only fields
(`GOMBIT_COOKIE_SECURE`, `GOMBIT_COOKIE_SAMESITE`) and their production
requirements.

`gombit new` writes a gitignored `.env` with a per-project random HMAC secret.
Generated `.env.example` keeps a short development placeholder for
documentation. Do not `cp .env.example .env` over an existing `.env` — that
replaces the per-project secret with the public placeholder. Production
rejects that value.

## Generated frontend

`frontend/src/auth/session.ts` holds both tokens in module variables.
`createAppClient` sends the access token and, on 401, rotates `/auth/refresh`
once. Concurrent 401s wait on that same refresh and retry instead of failing
with the stale response. The retry rebuilds the request from buffered body
bytes so POST/PATCH JSON is resent after silent refresh (buffering is gated
on method because Firefox's Request.body getter is unimplemented). `RequireAuth`
sends anonymous users to `/login`.
Logout clears memory and revokes the refresh token.

Product pages sit behind `RequireAuth`. Register from the login page is a
demo/bootstrap path, not a full identity product.

## Example

```sh
go run ./examples/auth
```

See [`examples/auth`](../examples/auth) and [`docs/frontend.md`](frontend.md).
