# Admin (ADMIN-1)

Gombit's Django-style admin is a **runtime generic admin** over an explicit
model registry and a Huma introspection + data-plane API
([ADR-013](adr/013-runtime-generic-admin.md)). This issue ships the registry
and HTTP contract. The framework-owned React SPA under `/admin/` is ADMIN-2
and is **not** in this tree yet.

| Issue | What it ships |
| --- | --- |
| ADMIN-0 (#33) | ADR-013. Done. |
| ADMIN-1 (#34) | `admin.Register` + `GET /api/v1/admin/meta` + generic `/api/v1/admin/resources/{slug}` |
| ADMIN-2 | Framework-owned SPA under `/admin/` — not here |
| ADMIN-3 | Groups/permissions on top of `IsSuperuser` — not here |

## When routes mount

`framework.New` mounts empty admin Huma routes **only** when cookie session
auth is on (`cfg.Auth.Mode == cookie` / `AuthModeCookie`) and a database is
attached. JWT-only apps do not get `/api/v1/admin/…` (they 404; the paths
are absent from OpenAPI). Dual Bearer-API + cookie-admin in one process is
not introduced here.

Session is required. Until ADMIN-3, `auth.User.IsSuperuser` is the only
admin principal (`gombit createsuperuser`). `/auth/register` never sets
that flag.

| Caller | Result |
| --- | --- |
| Anonymous | D10 `authentication` (401) |
| Authenticated non-superuser | D10 `authorization` (403) |
| Superuser, disabled action | D10 `authorization` (403) |
| Superuser, unknown slug or id | D10 `not_found` (404) |

CSRF on POST/PATCH/DELETE is the existing M5-3 global middleware
(`X-CSRF-Token`). See [auth-cookie.md](auth-cookie.md).

## Registration

Feature packages register models explicitly — typically from
`internal/<feature>/admin.go` or `routes.go`. The framework never walks
GORM models, AutoMigrate lists, or packages.

```go
func RegisterAdmin(app *framework.App) error {
    return admin.Register(app, Product{}, admin.Options{
        Slug:     "products",
        Singular: "Product",
        Plural:   "Products",
        Fields: []admin.Field{
            {Name: "id", Type: admin.TypeInteger, ReadOnly: true},
            {Name: "name", Type: admin.TypeString, Required: true},
            {
                Name: "category_id",
                Type: admin.TypeRelation,
                Related: &admin.Relation{
                    Slug:       "categories",
                    Kind:       admin.RelBelongsTo,
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
        Permissions: admin.Permissions{
            View:   "admin.products.view",
            Create: "admin.products.create",
            Update: "admin.products.update",
            Delete: "admin.products.delete",
        },
    })
}
```

`Options` is the source of truth. Missing or duplicate `Slug` is an error at
`Register`. After `Register` returns, handlers read stored values plus a
constructor for `T` (`Register[T any]`). They do **not** reflect over
arbitrary Go types.

### Fields, PK, and names

- **`Field.Name`** is the JSON object key in meta and in data-plane rows.
  For v1, `Name` is also the GORM/SQL column unless `Field.Column` is set
  (use that when the Go exported name or GORM column differs).
- **`Options.PK`** is the JSON/field name of the primary key. Empty means
  derive the GORM primary key **at Register** and store it. The PK field
  must appear in `Fields`.
- **Empty `Fields`** derives a default from the struct once, inside
  `Register`, via `admin.FieldsFrom(T)`. That helper may use `reflect`
  **only at registration time**. Do not call it from request handlers.
- **`created_at` / `updated_at`** are implicit GORM timestamp columns.
  They may appear in `List` and `Ordering` even when omitted from
  `Fields`. When the model has those columns, list and detail row JSON
  include the values. Search and filter still require an explicit field.
  If the model has no such GORM column, `Register` errors.
- Zero `Actions` defaults to all enabled. Empty `Permissions` default to
  `admin.{slug}.{view,create,update,delete}` and are echoed in meta only
  (ADMIN-3 enforces them).

Closed field types: `string`, `text`, `integer`, `float`, `decimal`,
`boolean`, `datetime`, `date`, `uuid`, `json`, `relation`.

Relation `kind` is `belongs_to` or `has_many`. **`belongs_to`** is stored as
the foreign key on create/update. **`has_many` is meta-only** in ADMIN-1:
the data plane does not nest related collections and treats the field as
read-only. `Register` rejects `has_many` (and any field with no SQL column)
in Search, Filter, or Ordering.

`Register` does not AutoMigrate. The application still owns schema
(Atlas / `AutoMigrate` in examples).

Generated apps are **not** auto-registered. Prefer an explicit
`product.RegisterAdmin` in the app; this repo does not scaffold that call
so JWT goldens stay still.

## HTTP (Huma on `app.API()`, D10)

Paths honor `config.API.Prefix` (default `/api/v1`) and appear in OpenAPI.

| Method | Path | Body |
| --- | --- | --- |
| `GET` | `/api/v1/admin/meta` | `{ "data": { "models": [ ... ] }, "meta"?: { "auth": { "mode": "cookie", "bootstrap": "is_superuser" } } }` |
| `GET` | `/api/v1/admin/meta/{slug}` | `{ "data": { /* one model */ } }` — 404 `not_found` if unknown |
| `GET` | `/api/v1/admin/resources/{slug}` | list; `meta` is `contract.PageMeta` (`page`, `per_page`, `total`) |
| `POST` | `/api/v1/admin/resources/{slug}` | create writable fields |
| `GET` | `/api/v1/admin/resources/{slug}/{id}` | detail |
| `PATCH` | `/api/v1/admin/resources/{slug}/{id}` | update writable fields |
| `DELETE` | `/api/v1/admin/resources/{slug}/{id}` | `{ "data": { "ok": true } }` |

`data.models` is required on the catalog (empty array when nothing is
registered). Raw `*gin.Engine` is **not** used for these endpoints.

List query parameters:

- `page`, `per_page` (default page 1, per_page 20, max 100)
- `search` (OR `LIKE` across `Options.Search`)
- `ordering` (a field from `Options.Ordering`; prefix `-` for DESC)
- one query key per `Options.Filter` field

Row JSON is keyed by registered field names, plus implicit `created_at` /
`updated_at` when the GORM model has them and they were omitted from
`Fields`. Create/update accept only writable (non-readonly) fields.
Unknown keys, readonly keys, and type failures render D10
`validation_error` with `fields`.

The application's public CRUD API stays the feature's own typed Huma
routes. Admin does not replace them.

## Example

[`examples/admin`](../examples/admin) — cookie mode + SQLite +
`admin.Register` of `widget.Widget`, plus curl for meta and one CRUD
cycle. Superuser is seeded with `auth.Service.CreateSuperuser` (same path
as `gombit createsuperuser`).

## Out of scope

- ADMIN-2 React SPA / `/admin/` embed
- Groups/permissions tables (ADMIN-3); permission **keys** only
- `--admin` generator / golden template changes
- M6 batteries (jobs, events, scheduler, mail, storage, gRPC, multi-tenancy, i18n)
- `localStorage` tokens
