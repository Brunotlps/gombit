# Production build and frontend embedding

Split deploy is the **default** (build plan C5). The Vite app under
`frontend/` is a separate origin (or CDN) in production. Embedding the
frontend into the Go binary is **opt-in**.

Django's `collectstatic` is folded into `gombit build --embed` — there is
no separate collectstatic command.

See also [cli.md](cli.md#gombit-build---embed) and [frontend.md](frontend.md).

## Default: split

```text
Static frontend / CDN
        ↓
      browser
        ↓
Go API service
```

After `gombit new`, `go run ./cmd/server` serves the API, probes, and
OpenAPI without a Vite `dist`. The scaffold always wires
`framework.WithEmbeddedFrontend`, but the placeholder `internal/web/static`
tree has only `.keep` (no `index.html`), so the runtime does not install
SPA fallback. Unknown paths stay 404.

`VITE_API_URL` is public. For a split deploy, set it to the API **origin
only** (OpenAPI paths already include `/api/v1`). Never put JWT secrets in
`VITE_*`. Access tokens stay in memory.

## Opt-in: `gombit build --embed`

```text
Vite production build
   ↓
frontend/dist
   ↓
collectstatic → internal/web/static
   ↓
go:embed + go build ./cmd/server
   ↓
single binary
```

```sh
gombit build --embed
gombit build --embed --out bin/server
gombit build --embed --dry-run
```

A bare `gombit build` without `--embed` is refused. That is deliberate: v0.1
must not silently change the default production path to embed.

The compiled binary serves:

| Request | Result |
| --- | --- |
| Huma `/api/*`, `/openapi.json`, `/docs` | API / contract (not `index.html`) |
| `/livez`, `/readyz`, `/metrics` | Probes |
| GET `/assets/…` when the file exists in the embed FS | That file, with a sane Content-Type |
| GET `/`, `/login`, `/products/new`, other missing frontend paths | `index.html` (SPA fallback) |
| Non-GET unmatched routes | 404 — not `index.html` |

Path names are cleaned; `..` cannot escape the embed FS.

## SPA Content-Security-Policy

Global middleware sets `Content-Security-Policy: default-src 'self'` on every
response. When the embedded frontend serves `index.html` (GET `/` and SPA
fallback), that header is overwritten so `--ui mui` + `--embed` can load
Roboto and Emotion-injected `<style>` tags:

- `script-src 'self'` — Vite production JS is hashed same-origin modules
  (no `'unsafe-inline'` scripts)
- `style-src 'self' 'unsafe-inline' https://fonts.googleapis.com` — Emotion
  runtime styles plus the Google Fonts stylesheet
- `font-src 'self' https://fonts.gstatic.com` — Roboto font files
- `connect-src 'self'` — same-origin API
- `img-src 'self' data:`

JSON API, probes, and `/metrics` keep `default-src 'self'`. `/docs` (when
enabled) keeps Huma's Swagger UI policy.

## Scaffold hook

`gombit new` writes:

```
internal/web/embed.go     //go:embed all:static; web.FS()
internal/web/static/.keep // placeholder so embed compiles; no index.html
```

`cmd/server` always passes `framework.WithEmbeddedFrontend(web.FS())`.
Collectstatic output lives in `internal/web/static/` — not `dist/`, which
goldens and `.gitignore` already skip. Generated `.gitignore` keeps Vite
output out of git:

```
internal/web/static/*
!internal/web/static/.keep
```

(`frontend/dist` is already ignored via `dist/`.)

## Example

[`examples/embed`](../examples/embed) mounts a tiny `embed.FS` with
`index.html` + `assets/app.js` and a Huma API route, proving API + static +
index fallback.
