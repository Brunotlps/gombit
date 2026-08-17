# OpenAPI emission and docs

Gombit emits OpenAPI 3.1 from Huma-typed handlers. The live document is served
at `/openapi.json`. A FastAPI-style interactive docs UI is served at `/docs`
in local and test environments. `gombit openapi generate` writes that document
to disk.

## Live surfaces

| Surface | URL | Role |
| --- | --- | --- |
| OpenAPI document | `/openapi.json` | Machine-readable contract (CLI, CI, codegen) |
| Interactive docs | `/docs` | Swagger UI backed by `/openapi.json`, with try-it-out against the same server |

`examples/contract` registers Huma widget routes. After `go run ./examples/contract`:

```sh
curl -sS http://127.0.0.1:8080/openapi.json
# open http://127.0.0.1:8080/docs
```

Try-it-out requests hit the running app, so D10 validation and category errors
appear as they would for any client. The docs page uses Huma's Swagger UI
renderer and loads the UI assets from `unpkg.com`; Huma sets a page-specific
CSP (`connect-src 'self'`) so try-it-out can call the same origin. Other
routes keep the runtime `default-src 'self'` policy.

Raw `app.Router()` routes (webhooks, SSE, probes, metrics) stay out of the
OpenAPI document and therefore out of `/docs`.

## Production

Docs are on by default in `development` and `test`. `config.Load` turns them
off in `production` unless `GOMBIT_DOCS_ENABLED=true`. `/openapi.json` stays
available so codegen and drift checks still work.

| Variable | Field | Default |
| --- | --- | --- |
| `GOMBIT_DOCS_ENABLED` | `Config.API.DocsEnabled` | `true` outside production; `false` in production when unset |

Applications that pass `framework.WithConfig` set `API.DocsEnabled` explicitly.
`contract.HumaConfigFor(title, version, docsEnabled)` is the Huma helper
`framework.New` uses.

## Write the spec to disk

With the app running:

```sh
go run ./cmd/gombit openapi generate \
  --url http://127.0.0.1:8080/openapi.json \
  --out openapi.json
```

`--url` defaults to `http://127.0.0.1:8080/openapi.json`. `--out` defaults to
`openapi.json`. The CLI validates the document as OpenAPI 3.1 before writing.

In-process generation (tests, `go:generate`) uses the same helpers:

```go
spec, err := contract.OpenAPIJSON(app.API())
if err != nil {
    return err
}
return contract.WriteOpenAPI("openapi.json", app.API())
```

`contract.OpenAPIJSON` matches the bytes served at `/openapi.json`.

## What is not here yet

- TypeScript client generation (`gombit client generate`): M3-4
- CI drift check that fails when the spec is stale: M3-5
