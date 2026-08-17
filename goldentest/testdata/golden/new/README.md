# demo

Scaffolded by `gombit new`. Feature-package layout is build plan §3.2.

## Run

```sh
cp .env.example .env
go run ./cmd/server
```

The server listens on `GOMBIT_HTTP_ADDR` (default `:8080`). Public API
routes are under `/api/v1`. Interactive docs are at `/docs` when
`GOMBIT_DOCS_ENABLED` is on. The live OpenAPI 3.1 document is
`/openapi.json`.

For local development, one command runs the API, Vite HMR, and live
TypeScript client regeneration:

```sh
gombit dev
```

`gombit dev` prints a service table (Backend, Frontend, OpenAPI, API docs).
Backend reload uses `air` or `watchexec` when installed; otherwise it falls
back to `go run ./cmd/server`. The frontend uses `pnpm` when available,
otherwise `npm`. Node.js is required for Vite HMR. The Vite stub here is
enough to start HMR; the full React skeleton is M5-1.

This module requires [`github.com/LAA-Software-Engineering/gombit`](https://github.com/LAA-Software-Engineering/gombit).
After scaffolding, pin a released version:

```sh
go get github.com/LAA-Software-Engineering/gombit@latest
go mod tidy
```

The default module path is `github.com/example/demo` (override with
`--module`).

## Recorded choices (`gombit.yaml`)

| Key | Value | Notes |
| --- | --- | --- |
| database | `sqlite` | sqlite, postgres, or mysql |
| cache | `memory` | memory, redis, or noop |
| auth | `jwt` | Bearer JWT is the v0.1 default; cookie/CSRF is M5-3 |
| ui | `minimal` | minimal/headless default; MUI CRUD preset is M5-4 |

Runtime still reads `GOMBIT_*` environment variables, not `gombit.yaml`.

## Layout

- `cmd/server` — `config.Load`, `framework.New`, explicit `product.Register`
- `internal/product` — model, Huma handlers, routes (no `service.go` / `repo.go`)
- `internal/platform` — database open + AutoMigrate for the example product
- `database/migrations`, `database/seeds` — Atlas SQL (see `gombit db`)
- `frontend/` — minimal Vite + TypeScript stub for `gombit dev` (M5-1 owns the React skeleton)

## Migrations

```sh
gombit db makemigrations create_products \
  --model github.com/example/demo/internal/product.Product
gombit db migrate
gombit db status
```

Example product handlers use GORM AutoMigrate on startup so the list/create
API works before the first Atlas migration exists. Prefer `gombit db` for
schema you intend to keep.

## Add a resource

```sh
gombit make resource Book title:string:required
```

That writes `internal/book/` (model, thin Huma handler, routes), vanilla
TypeScript list/form pages, and registers `book.Register(app)` in
`cmd/server/main.go` via `go/ast`. Re-running is idempotent and will not
clobber edits unless you pass `--force`. `--service` / `--repo` are opt-in.

Frontend pages import types from `frontend/src/api/generated`. Run
`gombit client generate` or `gombit dev` after the API is up.

If Atlas is not installed, the GORM model is still ready for:

```sh
gombit db makemigrations create_books --model github.com/example/demo/internal/book.Book
```

