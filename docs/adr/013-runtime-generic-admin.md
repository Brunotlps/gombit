# ADR-013: Runtime Generic Admin over an Introspection API

## Status

Accepted.

## Context

Gombit's post-v0.1 flagship is a Django-style admin: register a model, get a
working staff UI. No Go web framework (Gin, Echo, Fiber, Encore) ships a real
one. Build plan M6-Admin is that surface. It is **not** part of the v0.1 CRUD
loop.

The generate-vs-runtime rule (build plan §3.3) decides the shape:

> Behavior lives in the versioned runtime. Generated code is a thin, one-time
> scaffold the user owns. The framework never rewrites user-owned files.

If admin screens were `--admin` / `gombit make resource --admin` pages copied
into every generated `frontend/`, `gombit upgrade` could not evolve them, each
app would own a fork of the UI, and ADMIN-2's acceptance criterion ("zero
per-model frontend code") would be impossible.

The other locked constraints this ADR must encode, not reopen:

- **Principle 6.2 (explicit Go, minimal magic):** no Django metaclasses, no
  request-time walking of arbitrary Go types, no reflection discovery of every
  GORM model / AutoMigrate list. ADMIN-1 is explicit
  `admin.Register(Model, opts)`.
- **C2:** generated apps stay feature-packages under `internal/<feature>/`. A
  feature registers its models from `routes.go` or an `admin.go` beside the
  model — never Laravel `app/models` / `app/admin`.
- **C1:** the public HTTP contract is Huma-typed and appears in OpenAPI 3.1.
  Success and error bodies are D10.
- **C3:** Bearer JWT remains the v0.1 **API** default (access token in memory,
  never `localStorage`). Session/cookie (`--auth cookie` /
  `AuthModeCookie`) is first-class and a **hard prerequisite** of the admin.
  The admin UI uses HttpOnly session cookies + CSRF (M5-3). Superuser
  (`User.IsSuperuser`, seeded by M4-6 `gombit createsuperuser`) is the
  bootstrap admin; groups/permissions are provided by ADMIN-3.
- **C4:** the default generated app UI stays minimal/headless. `--ui mui` is
  an opt-in preset for **application** screens, not the admin.
- **M5-5:** optional `go:embed` + SPA fallback is the pattern for serving a
  framework- or app-owned React bundle from Gin without putting those routes
  in OpenAPI.
- **Scope guard:** jobs, events, scheduler, mail, storage, gRPC, multi-tenancy,
  and i18n stay out. This decision did not itself implement ADMIN-1,
  ADMIN-2, or ADMIN-3.

M5-3 (issue #30) is done, so the session/CSRF prerequisite is met. ADMIN-0's
job is to lock the registry/introspection contract and the "framework-owned
SPA" decision so ADMIN-1 is not blocked.

Names below (`admin.Register`, `/api/v1/admin/meta`, field type strings) are
**provisional**. They are documentation for implementers. This ADR does not
add a runtime `admin` package.

## Decision

Gombit's admin is a **runtime generic admin** driven by an **explicit model
registry** and a **Huma introspection API**. A framework-owned React app
renders list/detail/create/edit/delete from that metadata. Generated apps do
not receive `--admin` pages.

### 1. Registration API (ADMIN-1)

A feature-package registers models **explicitly** at startup, typically from
`internal/<feature>/routes.go` or `internal/<feature>/admin.go`:

```go
// internal/product/admin.go — sketch, not a shipped API
func RegisterAdmin(app *framework.App) {
    admin.Register(app, Product{}, admin.Options{
        Slug:    "products",
        Singular: "Product",
        Plural:  "Products",
        Fields: []admin.Field{
            {Name: "id", Type: "uuid", ReadOnly: true},
            {Name: "name", Type: "string", Required: true},
            {
                Name: "category_id",
                Type: "relation",
                Related: &admin.Relation{
                    Slug:       "categories",
                    Kind:       "belongs_to",
                    LabelField: "name",
                },
            },
        },
        List:     []string{"name", "category_id"},
        Search:   []string{"name"},
        Filter:   []string{"category_id"},
        Ordering: []string{"name", "created_at"},
        Actions: admin.Actions{
            List: true, Detail: true, Create: true, Update: true, Delete: true,
        },
        // Permission keys are declared here and enforced by ADMIN-3.
        Permissions: admin.Permissions{
            View:   "admin.products.view",
            Create: "admin.products.create",
            Update: "admin.products.update",
            Delete: "admin.products.delete",
        },
    })
}
```

`Options` is the **source of truth** for what the introspection endpoint and
the generic data plane expose:

| Field | Role |
| --- | --- |
| `Slug` | URL key (`products`). Stable, lowercase, unique per app. |
| `Singular` / `Plural` | UI labels. |
| `Fields` | Concrete field list: `name`, `type`, `required`, `readonly`, optional `related`. |
| `List` | Columns on the list view. |
| `Search` | Fields the list `search` query param applies to. |
| `Filter` | Fields the list may filter on. |
| `Ordering` | Fields the list may order by. |
| `Actions` | Which of list / detail / create / update / delete are enabled. |
| `Permissions` | Stable keys enforced by ADMIN-3. ADMIN-1 stores them on the registry and echoes them in meta. |

Closed field-type set for v1 of the admin (ADMIN-1 may add members with a
docs bump, not a parallel type system):

`string`, `text`, `integer`, `float`, `decimal`, `boolean`, `datetime`,
`date`, `uuid`, `json`, `relation`.

Relation `kind` is `belongs_to` or `has_many` for the first cut. Many-to-many
is deferred until a later admin issue needs it.

**How fields get onto the registry without request-time reflection:**

- Callers pass `Options.Fields` explicitly. That list is what HTTP handlers
  read.
- `admin.Register` may offer a **registration-time** convenience, e.g.
  `admin.FieldsFrom(Product{})` or "empty `Fields` derives a default from the
  struct once," analogous to passing a model into GORM. That helper may use
  `reflect` **only inside `Register`**, on the one type being registered, to
  produce a concrete `[]Field` stored on the registry. It must be documented
  as registration-time only and covered by tests in ADMIN-1.
- After `Register` returns, the registry holds values (slug, field slice,
  action flags, a constructor `func() any` / generic `Register[T]` so GORM
  can `Create`/`Find` that type). Request handlers **must not** walk
  arbitrary Go types, scan `AutoMigrate` lists, or discover models by
  package crawl.

Missing `Slug` is an error at `Register`. Duplicate slugs are an error at
`Register`. Unregistered models do not appear in admin.

### 2. Introspection HTTP (ADMIN-1)

Mounted on `app.API()` as Huma operations under `config.API.Prefix`
(default `/api/v1`). They appear in OpenAPI 3.1. D10 envelopes.

| Method | Path | Body |
| --- | --- | --- |
| `GET` | `/api/v1/admin/meta` | `{ "data": { "models": [ ... ] }, "meta"?: { ... } }` |
| `GET` | `/api/v1/admin/meta/{slug}` | `{ "data": { /* one model */ } }` — 404 D10 `not_found` if unknown |

Both require a cookie session (C3). ADMIN-3 enforces the registered model's
view/create/update/delete keys through direct or group grants.
`User.IsSuperuser` bypasses every permission check. Unauthenticated → D10
`authentication` (401). Authenticated without the required permission → D10
`authorization` (403). Mutating admin routes additionally go through the
existing M5-3 CSRF middleware (`X-CSRF-Token`).

Catalog `data` sketch (normative field names for ADMIN-1; see
[`testdata/admin-meta.json`](testdata/admin-meta.json)):

```json
{
  "data": {
    "models": [
      {
        "slug": "products",
        "singular": "Product",
        "plural": "Products",
        "pk": "id",
        "fields": [
          {"name": "id", "type": "uuid", "required": false, "readonly": true},
          {"name": "name", "type": "string", "required": true, "readonly": false},
          {
            "name": "category_id",
            "type": "relation",
            "required": false,
            "readonly": false,
            "related": {"slug": "categories", "kind": "belongs_to", "label_field": "name"}
          }
        ],
        "list": ["name", "category_id"],
        "search": ["name"],
        "filter": ["category_id"],
        "ordering": ["name", "created_at"],
        "actions": {
          "list": true,
          "detail": true,
          "create": true,
          "update": true,
          "delete": true
        },
        "permissions": {
          "view": "admin.products.view",
          "create": "admin.products.create",
          "update": "admin.products.update",
          "delete": "admin.products.delete"
        },
        "can": {
          "view": true,
          "create": false,
          "update": false,
          "delete": false
        }
      }
    ]
  },
  "meta": {
    "auth": {"mode": "cookie", "bootstrap": "permissions"}
  }
}
```

`meta.auth` on the catalog tells the SPA which authorization rule is in
force. ADMIN-3 adds per-user `can` flags and filters the catalog to models
the caller may view. A regular user with no view permission receives 403;
a superuser may still receive an empty catalog when nothing is registered.

Raw `*gin.Engine` (`app.Router()`) is **not** used for these endpoints. The
escape hatch is reserved for the admin SPA static files and `/admin/`
fallback, which must stay **out** of OpenAPI (same reason M5-5 SPA fallback
is raw Gin).

### 3. CRUD data plane (ADMIN-1 / ADMIN-2)

The React admin must have **zero per-model frontend code** (ADMIN-2 AC).
Therefore the runtime, not each feature-package, implements a **generic
admin data plane** from the registry:

| Method | Path |
| --- | --- |
| `GET` | `/api/v1/admin/resources/{slug}` |
| `POST` | `/api/v1/admin/resources/{slug}` |
| `GET` | `/api/v1/admin/resources/{slug}/{id}` |
| `PATCH` | `/api/v1/admin/resources/{slug}/{id}` |
| `DELETE` | `/api/v1/admin/resources/{slug}/{id}` |

These are Huma-typed operations on `app.API()`, session-required, CSRF on
unsafe methods, D10 envelopes. List uses `contract.PageMeta`
(`page`, `per_page`, `total`). List query parameters ADMIN-1 should honor:

- `page`, `per_page`
- `search` (applied to `Options.Search`)
- `ordering` (a field from `Options.Ordering`, optional `-` prefix for DESC)
- one query key per `Options.Filter` field

Create/update bodies are JSON objects of writable fields. Unknown slugs,
disabled actions, and unknown ids are D10 `not_found` or `authorization` as
appropriate. Validation failures use D10 `validation_error` with `fields`.

Row payloads are JSON objects keyed by the registered field names. The
OpenAPI schema for a row is necessarily generic (`object`); that is
accepted. The application's **public** CRUD API stays the feature's own
typed Huma routes. Admin does not replace those routes and does not require
the feature to write a second handler set.

**Why not reuse per-model public Huma routes?** The SPA would then need a
per-slug client and would break ADMIN-2's "zero per-model frontend code"
rule. Public DTOs also omit admin-only columns and use different auth
(Bearer by default). A registry-driven data plane keeps one client in the
admin SPA and one authorization story (session + superuser / ADMIN-3).

ADMIN-1 implements meta + the data plane. ADMIN-2 consumes them. This ADR
does not ship either.

### 4. Admin SPA (ADMIN-2)

The admin UI is a **framework-owned** Vite + React + TypeScript app, versioned
with the Gombit module (path TBD in ADMIN-2; e.g. `internal/adminui`). It is
**not** copied into generated `frontend/`.

Serving follows M5-5: the built assets live in a framework `embed.FS` and
are mounted on raw Gin with SPA fallback under **`/admin/`** (assets under
`/admin/assets/…`). Those GET routes stay out of OpenAPI. API calls go to
`/api/v1/admin/…` and the existing cookie auth endpoints
(`/api/v1/auth/csrf`, `/auth/login`, `/auth/logout`, `/me`) on the same
origin, so HttpOnly cookies attach and CSRF double-submit works as in
[`docs/auth-cookie.md`](../auth-cookie.md).

The admin SPA may use MUI internally. That does **not** couple generated apps
to `--ui mui` (C4). Default app screens stay minimal; `--ui mui` remains the
application-CRUD preset.

Auth tokens never go in `localStorage` / `sessionStorage`. Cookie mode
already keeps access/refresh in HttpOnly cookies; the admin SPA may hold
only the CSRF token in memory (same as the generated cookie frontend).

`gombit new` does not scaffold admin pages. `gombit make resource` does not
gain `--admin`. A later issue may AST-append an `admin.Register` call in the
feature package; it still must not emit React pages.

Optional later (not ADMIN-2 AC): `gombit dev` prints an Admin URL in the
service table. Out of this ADR.

Mounting: introspection, data plane, and `/admin/` SPA mount only when
`Config.Auth.EffectiveMode() == AuthModeCookie` and auth is enabled.
Bearer-only apps keep the v0.1 API default and do not grow an admin. Dual
Bearer-API + cookie-admin in one process is not introduced here; cookie mode
is already mutually exclusive with JWT mode (M5-3). Apps that want admin use
`--auth cookie`.

### 5. Rejected alternatives

- **`--admin` / `gombit make resource --admin` generated CRUD pages.**
  Violates §3.3; forks the UI per app; fails ADMIN-2's zero per-model
  frontend code.
- **Deep reflection over all GORM models or the AutoMigrate list.**
  Violates principle 6.2; would expose models the app never opted into
  admin; request-time type walking is the Django-metaclass path Gombit
  refused.
- **Laravel-style `app/admin` (or `app/models`) tree.** Violates C2.
  Registration lives in the feature package.
- **Making `--ui mui` generated screens the admin.** Violates C4. MUI
  application screens are a preset for the user's app, not the staff UI.
- **Per-model admin Huma handlers written by each feature.** Duplicates the
  public API, forces per-model frontend code, and blocks a single generic
  SPA.

## Consequences

- ADMIN-1 implements `admin.Register`, the in-memory registry, Huma
  `GET /api/v1/admin/meta` (+ `/{slug}`), and the generic
  `/api/v1/admin/resources/{slug}` data plane. No request-time reflection.
  Tests cover registration errors, meta JSON, session/superuser gates, and
  CRUD against a registered fixture model (SQLite + PostgreSQL + MySQL when
  the data plane touches the DB).
- ADMIN-2 implements the framework-owned React app, `embed.FS` serving under
  `/admin/`, and screens driven only by meta + the generic data plane.
- ADMIN-3 enforces direct and group permissions using the keys stored on the
  registry. `IsSuperuser` bypasses every permission check. `/auth/register`
  still never sets `IsSuperuser`.
- Generated `frontend/` and `--ui mui` are unchanged by this ADR. No golden
  template updates belong in ADMIN-0.
- Agents must not re-litigate "generated admin pages vs runtime SPA," must
  not add `app/admin`, and must not walk GORM models without `Register`.
- No jobs, events, scheduler, mail, storage, gRPC, multi-tenancy, or i18n
  work hides under "admin."

## References

- Issue #33: `[ADMIN-0] ADR-013: runtime generic admin over an introspection API`
- Issue #30: `[M5-3] Cookie/session + CSRF preset (--auth cookie)` (prerequisite)
- Issue #34: `[ADMIN-1] Model registry + introspection endpoint`
- Issue #35: `[ADMIN-2] Generic React admin app`
- Issue #36: `[ADMIN-3] Admin auth + authorization`
- Issue #26: `[M4-6] gombit createsuperuser` (`IsSuperuser` bootstrap)
- Issue #32: `[M5-5] Optional go:embed build` (SPA embed pattern)
- Build plan §3.3, C1–C4, M6-Admin (`docs/GOMBIT_BUILD_PLAN.md`)
- Design doc §6.2 (explicit Go, minimal magic) — rationale only
- [`docs/admin.md`](../admin.md)
- [`docs/auth-cookie.md`](../auth-cookie.md)
- [`docs/build.md`](../build.md)
- [`docs/adr/011-contract-layer-huma.md`](011-contract-layer-huma.md)
- Fixture: [`testdata/admin-meta.json`](testdata/admin-meta.json)
