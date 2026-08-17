# Bearer JWT auth

Gombit's v0.1 API default is **Bearer JWT with refresh rotation** (C3 / D3).
Cookie/session + CSRF is a first-class mode but a separate issue ([M5-3]).
`gombit createsuperuser` is [M4-6].

Behavior lives in the `auth` runtime package. `framework.New` mounts the
Huma routes when `GOMBIT_JWT_SECRET` is set **and** a database is attached.
Generated apps (`gombit new --auth jwt`, the default) wire this through
`config.Load` + `framework.WithDatabase`; they do not copy handler code.

## Token storage (SPA)

| Token | Where | Notes |
| --- | --- | --- |
| Access JWT | **Memory only** | `Authorization: Bearer`. Never `localStorage` / `sessionStorage`. Lost on refresh, which is intended. |
| Refresh token | **Memory only** (JSON body) | Returned once on login/refresh. Rotated on each successful refresh. Not a cookie in v0.1. |

An HttpOnly refresh cookie is compatible with the Bearer *access* default, but
this milestone keeps the refresh token in the JSON body so M5-3 can add
cookie/CSRF without mixing modes. Do not put tokens in `VITE_*`.

## Endpoints

All paths use `config.API.Prefix` (default `/api/v1`). D10 envelopes.

| Method | Path | Auth |
| --- | --- | --- |
| `POST` | `/auth/register` | Public. Seeds a user (email + password). Until [M4-6]. |
| `POST` | `/auth/login` | Public. Returns `access_token`, `refresh_token`, `token_type`, `expires_in`. |
| `POST` | `/auth/refresh` | Public. Body `{ "refresh_token" }`. Issues a new pair; the old refresh token is invalid. |
| `POST` | `/auth/logout` | Public. Body `{ "refresh_token" }`. Revokes that refresh token. Bound access JWTs then fail. |
| `GET` | `/me` | Bearer access JWT. Example protected route for E2E. |

Passwords are hashed with bcrypt. Access JWTs are HS256, bound to the refresh
row so logout (and reuse of a rotated refresh token) 401s `/me`. Reusing a
revoked refresh token revokes that user's remaining refresh tokens.

## Config

| Variable | Field | Default |
| --- | --- | --- |
| `GOMBIT_JWT_SECRET` | `Config.Auth.JWTSecret` | empty (auth unmounted) |
| `GOMBIT_JWT_ACCESS_TTL` | `Config.Auth.AccessTokenTTL` | `15m` |
| `GOMBIT_JWT_REFRESH_TTL` | `Config.Auth.RefreshTokenTTL` | `168h` (7 days) |

Production rejects a **non-empty** JWT secret shorter than 32 characters at
`config.Load` / `Validate` and `gombit doctor` (Appendix C). The secret is
redacted by `Config.Redacted()` and `gombit config show`. Empty in production
leaves Bearer auth off; set a long secret to enable it.

Generated `.env.example` has a development placeholder. It is not a real
secret. Copy to `.env` and replace it before production.

## Generated frontend

`frontend/src/auth/session.ts` holds both tokens in module variables.
`createAppClient` sends the access token and, on 401, calls `/auth/refresh`
once. `RequireAuth` sends anonymous users to `/login`. Logout clears memory
and revokes the refresh token.

Product pages sit behind `RequireAuth`. Register from the login page is a
demo/bootstrap path, not a full identity product.

## Example

```sh
go run ./examples/auth
```

See [`examples/auth`](../examples/auth) and [`docs/frontend.md`](frontend.md).
