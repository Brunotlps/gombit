# Logging

M1-6 introduces the runtime logging surface. Gombit uses Zap for structured
logs and defaults to stderr JSON output without requiring MongoDB.

## Configuration

`config.Default()` enables:

- level: `info`
- sink: `stderr`

`GOMBIT_LOG_LEVEL` accepts `debug`, `info`, `warn`, and `error`.
`GOMBIT_LOG_SINK` accepts `stderr`, `stdout`, and `mongo`.

The `mongo` sink is a selectable external module hook. The runtime does not
import or open a MongoDB client. Applications that want Mongo-backed logs supply
a Zap core from their Mongo logging module and pass the resulting logger through
`framework.WithLogger`.

## Runtime Use

`framework.New` builds a default logger from `Config.Logging` when no logger is
provided. `app.Logger()` returns the `*zap.Logger` escape hatch:

```go
app, err := framework.New()
if err != nil {
	return err
}

app.Logger().Info("started")
```

HTTP-only apps and apps without Mongo configuration boot with the default
stderr logger.
