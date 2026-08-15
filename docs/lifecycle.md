# Application Lifecycle

The `framework` package owns the application lifecycle introduced in M1-2.
Applications construct an `App`, register optional hooks, and run it through
`framework.Run`.

```go
app, err := framework.New()
if err != nil {
    return err
}

app.OnStart(func(ctx context.Context) error {
    return nil
})

app.OnStop(func(ctx context.Context) error {
    return nil
})

return framework.Run(app)
```

Start hooks run in registration order. Stop hooks run in reverse registration
order so cleanup unwinds deterministically. Stop hooks receive a bounded
shutdown context; the default timeout is 10 seconds.

`App.Router()` exposes the underlying `*gin.Engine` as the raw Gin escape hatch
required by ADR-011. Public API routes that belong in OpenAPI still need Huma
typed handlers in later contract work.

The M1-2 runtime owns the lifecycle skeleton from the design doc: typed config,
HTTP server construction, basic runtime probes, signal handling, graceful
shutdown, and dependency cleanup hooks. Database, migrations, cache, auth,
observability, middleware parity, and embedded frontend mounting remain scoped
to later backlog issues.
