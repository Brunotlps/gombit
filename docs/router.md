# Application-Owned Route Registration

M1-3 de-domains the router introduced in M1-2: `framework.New` builds a
`*gin.Engine` with zero knowledge of any application domain and mounts only
its own endpoints. Today that means `/livez`, `/readyz`, and `/metrics`; the
OpenAPI document joins later (M3).

Applications register their own routes directly against the escape hatch:

```go
app, err := framework.New()
if err != nil {
    return err
}

registerProductRoutes(app.Router())
registerAccountRoutes(app.Router())

return framework.Run(app)
```

Each `registerXRoutes` function is the runtime equivalent of a feature
package's `routes.go` (build plan §3.2 /
`.cursor/skills/create-feature/references/layout.md`): registration is
**explicit**, called from `main`, and never discovered through reflection
(principle 6.2). `examples/router` demonstrates this with two independent
toy route groups, `ping` and `echo`.

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
  -> request timeout
  -> feature group middleware (if any)
    -> feature handler
```

Canonical design order (draft §13.3) also includes CORS, body-size limit, **XSS HTML-tag sanitization of request input**, rate limiting, and auth context. XSS sanitization is a fundamental security default and is owned by **M1-8**; it is not yet installed on the default router.

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
headers, timeout, and (once M1-8 lands) XSS HTML sanitization middleware for
that router.

`framework/router_test.go`'s `TestDefaultRouterMountsOnlyFrameworkEndpoints`
and `TestApplicationOwnedRouteRegistrationComposesIndependently` cover this;
`framework/app_test.go`'s `TestDefaultRouterRecoversFromPanics` covers
Recovery still applying to an application-registered route.

## What is not here yet

This surface does not include CORS, rate limiting, XSS HTML sanitization, or
authentication middleware. Those are extracted with their own runtime
dependencies in later issues:

- rate limiting and the cache interface: M1-5
- Mongo-backed access logging: M1-6
- XSS HTML-tag sanitization of request input: M1-8
- auth middleware: M5

Until then, an application that needs one of these can add it directly via
`app.Router().Use(...)` or on its own route groups; the framework will not
silently reorder or override it.
