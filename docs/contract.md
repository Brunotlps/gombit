# Contract (Huma DTOs + envelope)

Gombit's public API contract conventions: Huma-typed handlers over Gin, D10
success/error envelopes, validation tags → structured `fields`, and draft §41
application error categories mapped centrally to HTTP status codes.

ADR-011 selected Huma as the contract layer. OpenAPI emission, `/docs`, and
`gombit openapi generate` are documented in [`docs/openapi.md`](openapi.md).
The TypeScript client is documented in [`docs/client.md`](client.md).

## Registering contract routes

`framework.New` mounts a Huma API on the same `*gin.Engine` as `app.Router()`.
Register public API operations on `app.API()`:

```go
app, err := framework.New()
if err != nil {
    return err
}

prefix := app.Config().API.Prefix // default /api/v1
huma.Register(app.API(), huma.Operation{
    OperationID: "create-widget",
    Method:      http.MethodPost,
    Path:        prefix + "/widgets",
}, handler)

return framework.Run(app)
```

Use `app.Router()` only for routes that must stay outside the OpenAPI contract
(webhooks, SSE, legacy). See [`docs/router.md`](router.md).

`examples/contract` shows create, list (with `meta`), get, and not-found.

## Success envelope

```go
type createWidgetOutput struct {
    Body contract.Data[Widget] // {"data": {...}}
}

type listWidgetsOutput struct {
    // Typed meta so OpenAPI emits PageMeta fields (not empty object).
    Body contract.DataMeta[[]Widget, contract.PageMeta]
}
```

`contract.PageMeta` is the v0.1 pagination meta for collection responses.
Pass a non-nil pointer when meta should appear; a nil `Meta` is omitted.
A non-nil zero `PageMeta` still serializes (JSON `omitempty` does not drop empty structs).

Generated list handlers (`gombit new` product, `gombit make resource`, and
`examples/tutorial`) honor `page` / `per_page` query parameters the same way
the admin data plane does: default page 1, per_page 20, max 100
(`contract.ClampPage`). `meta.total` is a separate `COUNT`; `data` is the
`LIMIT`/`OFFSET` slice, so `len(data)` never exceeds `meta.per_page`.

```go
Body: contract.DataMeta[[]Widget, contract.PageMeta]{
    Data: items,
    Meta: &contract.PageMeta{Page: page, PerPage: perPage, Total: total},
},
```

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 125
  }
}
```

## Request DTOs

Go structs are the source of truth. Prefer Huma tags — not a separate
`validate:"..."` layer and not hand-written OpenAPI files.

```go
type CreateWidgetBody struct {
    Name  string `json:"name" minLength:"1" maxLength:"80" doc:"Human-readable name"`
    Color string `json:"color,omitempty" maxLength:"30"`
}
```

Conventions:

- Nest the JSON body under an input field named `Body` (Huma binding).
- Put validation metadata on body fields (`minLength`, `maxLength`, `format`,
  `enum`, `required`, and the other Huma tags).
- Document fields with `doc` / `example` so they appear in OpenAPI.

## Validation → D10 `fields`

Invalid requests return HTTP **422** with:

```json
{
  "error": {
    "code": "validation_error",
    "message": "The request contains invalid fields.",
    "fields": {
      "name": ["expected required property name to be present"]
    },
    "request_id": "..."
  }
}
```

`contract.Install` (called from `framework.New`) replaces Huma's default RFC
9457 Problem Details errors with this envelope. `fields` keys are derived from
Huma `ErrorDetail.Location` values:

| Huma location | D10 field key |
| --- | --- |
| `body.name` | `name` |
| `body.items[0].tags` | `items[0].tags` |
| `query.limit` | `query.limit` |
| `path.widget-id` | `path.widget-id` |

When Huma omits a location for a missing required property, Gombit infers the
field name from messages like `expected required property name to be present`.

`request_id` comes from the runtime request-ID middleware
(`X-Request-Id` / `framework.GetRequestIDFromContext`).

Content-Type stays `application/json` (not `application/problem+json`).

## Application errors (§41 categories)

Services and handlers should return category errors, not ad-hoc status codes.
Constructors build a D10 `ErrorEnvelope` (`huma.StatusError`). Wrap with
`contract.WithContext` so `request_id` is filled:

```go
return nil, contract.WithContext(ctx, contract.NotFound("widget not found"))
```

| Category | D10 `error.code` | HTTP |
| --- | --- | --- |
| `validation` | `validation_error` | 422 |
| `authentication` | `authentication` | 401 |
| `authorization` | `authorization` | 403 |
| `not_found` | `not_found` | 404 |
| `conflict` | `conflict` | 409 |
| `rate_limited` | `rate_limited` | 429 |
| `dependency_unavailable` | `dependency_unavailable` | 503 |
| `internal` | `internal` | 500 |

Helpers: `Validation`, `Authentication`, `Authorization`, `NotFound`,
`Conflict`, `RateLimited`, `DependencyUnavailable`, `Internal`, plus
`New(category, message)`. Always wrap with `WithContext` in handlers so
`request_id` is set — constructors alone leave it empty.

Gin middleware may also emit D10 errors that are not §41 categories.
`contract.PayloadTooLarge` (`error.code` `payload_too_large`, HTTP 413) is
used by XSS JSON sanitizer buffering (see [`docs/router.md`](router.md));
cookie CSRF uses `Authorization` (403). Do not treat 413 as a handler-level
§41 category or add it to the table above.

`WithFields` attaches D10 `fields` without forcing the code to
`validation_error` (for example a `conflict` with a field detail). Huma tag
validation still uses the Install path and always yields `validation_error`.

The `validation_error` code is preserved from M3-1 / D10 (not renamed to
`validation`).

## OpenAPI

`/openapi.json` is served from the Huma API and reflects the D10
`ErrorEnvelope` schema. `/docs` is the FastAPI-style Swagger UI (on by default
in local/dev). `gombit openapi generate` writes the live spec to disk. CI
regenerates the sample spec and TypeScript client from `client.SampleApp()`
and fails on drift — see [`docs/openapi.md`](openapi.md) and
[`docs/client.md`](client.md).

## What is not here yet

- Pagination query DSL / filter/sort helpers (design §42)
- gRPC status mapping (post-v0.1)
