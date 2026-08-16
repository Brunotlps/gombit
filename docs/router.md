# Application-Owned Route Registration

M1-3 de-domains the router introduced in M1-2: `framework.New` builds a
`*gin.Engine` with zero knowledge of any application domain and mounts only
its own endpoints. Today that means `/livez`, `/readyz`, `/metrics`, and the
Huma OpenAPI preview routes (`/openapi.json` and siblings). Public API handlers
register on `app.API()` (see [`docs/contract.md`](contract.md)); raw Gin routes
continue to use `app.Router()`.

Applications register their own routes against the appropriate surface:

```go
app, err := framework.New()
if err != nil {
    return err
}

registerProductRoutes(app.API(), app.Config().API.Prefix) // contract / OpenAPI
registerWebhookRoutes(app.Router())                         // escape hatch

return framework.Run(app)
```

Each `registerXRoutes` function is the runtime equivalent of a feature
package's `routes.go` (build plan §3.2 /
`.cursor/skills/create-feature/references/layout.md`): registration is
**explicit**, called from `main`, and never discovered through reflection
(principle 6.2). `examples/router` demonstrates raw Gin groups; `examples/contract`
demonstrates Huma-typed registration with D10 validation errors.

## No module/registry abstraction

Gombit does not provide a `Module` type, a route registry, or any
composition layer beyond `*gin.Engine` itself. `app.Router()` already is the
idiomatic mechanism: `router.Group(prefix)` gives a feature its own route
group and its own group-scoped middleware, and two features registered this
way cannot interfere with each other. Adding a bespoke abstraction on top
would duplicate what Gin already does well and cut against the "no reflection
discovery" principle — see `framework/router_test.go`'s
`TestApplicationOwnedRouteRegistrationComposesIndependently`, which proves
two independently-registered groups compose without cross-module leakage.

## Middleware ordering

Framework middleware installed before `New` returns wraps every route
registered afterward, including application route groups and their own
group-scoped middleware. Order is:

```text
Recovery
  -> request ID
  -> trace context
  -> request metrics
  -> security headers
  -> XSS HTML-tag sanitization (request input)
  -> request timeout
  -> feature group middleware (if any)
    -> feature handler
```

XSS sanitization is a fundamental security default (M1-8): response headers
alone are not enough. The runtime strips HTML tags from JSON string fields
(POST/PUT/PATCH) and GET query values using a first-party sanitizer built on
`golang.org/x/net/html`. Fields named `password` are left unchanged. Invalid
JSON is passed through so Gin/Huma can return normal validation errors.

Canonical design order (draft §13.3) also includes CORS, body-size limit, rate
limiting, and auth context. Those remain deferred; when body-size lands it
inserts immediately before XSS.

Request IDs use the `X-Request-Id` header. If the caller provides one, the
runtime preserves it; otherwise it generates one and stores it on both Gin's
context and `c.Request.Context()` for downstream code:

```go
requestID := framework.GetRequestID(c)
requestIDFromContext := framework.GetRequestIDFromContext(c.Request.Context())
```

Trace context currently preserves the W3C `Traceparent` trace ID when present
and exposes the active trace ID through `X-Trace-Id`,
`framework.GetTraceID(c)`, and `framework.GetTraceIDFromContext(ctx)`. Full
OpenTelemetry exporter wiring remains future runtime work; M1-7 preserves the
runtime seam and parity tests.

`/metrics` exposes Prometheus text-format request counters, active request
gauge, and request-duration sums for the runtime router. The text renderer is
intentionally minimal for M1 runtime parity; a later observability issue can
swap in `prometheus/client_golang` or full OpenTelemetry exporter wiring
without changing the route contract. Trusted proxies are configured through
`config.Config.HTTP.TrustedProxies` or
`GOMBIT_HTTP_TRUSTED_PROXIES`; when unset, Gin ignores forwarded-client IP
headers and uses the direct TCP peer.

`framework.WithRouter` is the custom-router escape hatch. When an application
passes its own `*gin.Engine`, Gombit applies trusted-proxy configuration only;
the application owns recovery, request ID, trace context, metrics, security
headers, XSS HTML sanitization, and timeout middleware for that router.

`framework/router_test.go`'s `TestDefaultRouterMountsOnlyFrameworkEndpoints`
and `TestApplicationOwnedRouteRegistrationComposesIndependently` cover this;
`framework/app_test.go`'s `TestDefaultRouterRecoversFromPanics` covers
Recovery still applying to an application-registered route.

## What is not here yet

Contract DTOs, validation → D10 field errors, and `app.API()` are documented in
[`docs/contract.md`](contract.md). This router surface still does not include
CORS, rate limiting, or authentication middleware:

- auth middleware: M5
- `gombit openapi generate` CLI: M3-3

Until then, an application that needs middleware can add it directly via
`app.Router().Use(...)` or on its own route groups; the framework will not
silently reorder or override it.

`examples/router` posts JSON to `/echo` and returns the comment the handler
saw after XSS sanitization.
