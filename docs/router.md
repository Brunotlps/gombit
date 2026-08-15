# Application-Owned Route Registration

M1-3 de-domains the router introduced in M1-2: `framework.New` builds a
`*gin.Engine` with zero knowledge of any application domain and mounts only
its own endpoints. Today that means `/livez` and `/readyz`; metrics and the
OpenAPI document join later (M1-7, M3).

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

Framework middleware installed before `New` returns — currently
`gin.Recovery()` — wraps every route registered afterward, including
application route groups and their own group-scoped middleware. Order is:

```text
framework middleware (e.g. Recovery)
  -> feature group middleware (if any)
    -> feature handler
```

`framework/router_test.go`'s `TestDefaultRouterMountsOnlyFrameworkEndpoints`
and `TestApplicationOwnedRouteRegistrationComposesIndependently` cover this;
`framework/app_test.go`'s `TestDefaultRouterRecoversFromPanics` covers
Recovery still applying to an application-registered route.

## What is not here yet

This surface does not include CORS, security/XSS headers, rate limiting,
request-id/timeout, trusted-proxy configuration, or authentication
middleware. Those are extracted with their own runtime dependencies in later
issues:

- rate limiting and the cache interface: M1-5
- Mongo-backed access logging: M1-6
- request-id/timeout, security headers, trusted-proxy parity: M1-7
- auth middleware: M5

Until then, an application that needs one of these can add it directly via
`app.Router().Use(...)` or on its own route groups; the framework will not
silently reorder or override it.
