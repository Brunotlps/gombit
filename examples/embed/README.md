# Embedded frontend example

Minimal `framework.App` with `framework.WithEmbeddedFrontend` over a tiny
`go:embed` FS (`index.html` + `assets/app.js`) and a Huma API route.

Split deploy stays the default (C5). This example is the opt-in embed
serve path used by `gombit build --embed`. See [`docs/build.md`](../../docs/build.md).

## Run

```sh
go run ./examples/embed
```

## curl

GET `/`, `/login`, and `/assets/app.js` hit the embed FS (SPA fallback for
missing frontend routes). GET `/api/v1/ping` is the Huma API — not
`index.html`. `/readyz` stays the probe.

```sh
# SPA index
curl -sS http://127.0.0.1:8080/
curl -sS http://127.0.0.1:8080/login

# static asset
curl -sS http://127.0.0.1:8080/assets/app.js

# API (JSON, D10 envelope)
curl -sS http://127.0.0.1:8080/api/v1/ping

# probe
curl -sS http://127.0.0.1:8080/readyz

# unmatched POST is not index.html
curl -sS -o /dev/null -w '%{http_code}\n' -X POST http://127.0.0.1:8080/login
```

Interactive docs: [http://127.0.0.1:8080/docs](http://127.0.0.1:8080/docs).
