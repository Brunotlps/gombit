# Admin (post-v0.1)

Gombit's Django-style admin is the prioritized post-v0.1 flagship (build plan
M6-Admin). Implementation has not started.

The accepted architecture is **[ADR-013](adr/013-runtime-generic-admin.md)**:
a framework-owned React app over an explicit model registry and a Huma
introspection API. Not `--admin` scaffolded pages.

| Issue | What it ships |
| --- | --- |
| ADMIN-0 (#33) | This decision (ADR-013). Done when the ADR is merged. |
| ADMIN-1 | `admin.Register` + `GET /api/v1/admin/meta` + generic `/api/v1/admin/resources/{slug}` |
| ADMIN-2 | Framework-owned SPA under `/admin/` |
| ADMIN-3 | Groups/permissions on top of `IsSuperuser` |

Do not add a runtime `admin` package, admin HTTP routes, or an admin React
app until ADMIN-1. Session/cookie auth (`--auth cookie`) is a hard
prerequisite; see [auth-cookie.md](auth-cookie.md).
