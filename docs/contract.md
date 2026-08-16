# Contract (Huma DTOs + validation)

M3-1 introduces Gombit's public API contract conventions: Huma-typed handlers
over Gin, validation tags on request structs, and the locked D10 error envelope
with structured `fields`.

ADR-011 selected Huma as the contract layer. This document is the application-
facing guide for DTO shape and validation errors. Application error categories
and success `meta` belong to M3-2; OpenAPI CLI generation belongs to M3-3.

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

`examples/contract` shows a minimal create endpoint.

## Request / response DTOs

Go structs are the source of truth. Prefer Huma tags — not a separate
`validate:"..."` layer and not hand-written OpenAPI files.

```go
type CreateWidgetBody struct {
    Name  string `json:"name" minLength:"1" maxLength:"80" doc:"Human-readable name"`
    Color string `json:"color,omitempty" maxLength:"30"`
}

type createWidgetInput struct {
    Body CreateWidgetBody
}

type createWidgetOutput struct {
    Body contract.Data[Widget]
}
```

Conventions:

- Nest the JSON body under an input field named `Body` (Huma binding).
- Put validation metadata on body fields (`minLength`, `maxLength`, `format`,
  `enum`, `required`, and the other Huma tags).
- Wrap success payloads in `contract.Data[T]` so responses are `{"data": ...}`.
  Pagination `meta` arrives in M3-2.
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

## OpenAPI preview

With the default Huma config, `/openapi.json` is served for local inspection and
reflects the D10 `ErrorEnvelope` schema. `gombit openapi generate` and CI drift
checks land in M3-3 / M3-5.

## What is not here yet

- Central application error categories and HTTP status mapping (M3-2)
- Success envelope `meta` helpers (M3-2)
- `gombit openapi generate` / `gombit client generate` (M3-3 / M3-4)
