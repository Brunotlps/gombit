# Gombit CLI

`gombit` is a Cobra command tree (D13 / [ADR-014](adr/014-cli-cobra.md)).
Root help lists the command families. Nested families attach with
`AddCommand`; later app-registered management commands (M4-7) will use the
same mechanism.

```sh
go run ./cmd/gombit --help
```

## Command families

| Family | Role | Milestone |
| --- | --- | --- |
| `gombit new` | Scaffold an application | M4-1 |
| `gombit dev` | Run API + Vite with HMR and live client regen | M4-2 |
| `gombit make resource` | Generate a feature-package resource (AST-safe) | M4-3 |
| `gombit db …` | Atlas-backed migrations | M2, migrated onto Cobra in M4-1 |
| `gombit openapi generate` | Write the live OpenAPI 3.1 document | M3-3 |
| `gombit client generate` / `check` | TypeScript client + drift | M3-4, M3-5 |

Not in this milestone: `routes` / `doctor` / `config show` (M4-4),
`createsuperuser` (M4-6), `make command` (M4-7). Golden tests for generators
are M4-5; this command still has unit tests.

## `gombit new`

Non-interactive (the acceptance criterion):

```sh
gombit new demo --database sqlite
```

That writes a compiling app under `./demo`. The default Go module path is
`github.com/example/demo`. Override it with `--module`. After scaffolding,
pin the framework version:

```sh
cd demo
go get github.com/LAA-Software-Engineering/gombit@latest
go mod tidy
go run ./cmd/server
```

When developing against a local checkout of Gombit, add a replace in the
generated `go.mod` (do not commit a machine-specific path):

```
replace github.com/LAA-Software-Engineering/gombit => /path/to/gombit
```

### Flags

| Flag | Values | Default |
| --- | --- | --- |
| `--database` | `sqlite`, `postgres`, `mysql` | `sqlite` |
| `--cache` | `memory`, `redis`, `noop` | `memory` |
| `--auth` | `jwt`, `cookie`, `none` | `jwt` |
| `--ui` | `minimal`, `mui` | `minimal` |
| `--module` | Go module path | `github.com/example/<name>` |
| `--dry-run` | | print the file list without writing |
| `--force` | | required to write into a non-empty destination |

`--auth cookie` and `--ui mui` are recorded in `gombit.yaml` only. Cookie/CSRF
is M5-3; the MUI CRUD preset is M5-4. The generated `frontend/` directory is a
minimal Vite + TypeScript stub so `gombit dev` can start HMR; M5-1 owns the
full React skeleton (router, React Hook Form, auth pages).

If the project name is omitted and stdin is a TTY, `gombit new` prompts for
name and the choices above. Tests and CI pass flags so the command never
hangs.

Generators are additive: `--dry-run` writes nothing; a non-empty destination
is refused unless `--force` is set. `--force` overwrites scaffold files and
leaves other files in the destination alone.

### Layout

The scaffold matches build plan §3.2:

```
demo/
├── cmd/server/main.go
├── internal/
│   ├── platform/
│   └── product/          # model, handler.go, routes.go
├── database/migrations/
├── database/seeds/
├── config/
├── frontend/             # Vite stub (package.json, vite.config.ts, src/main.ts, src/resources.ts)
├── gombit.yaml
├── .air.toml
├── .env.example
├── go.mod
└── README.md
```

`cmd/server/main.go` calls `config.Load()`, `framework.New`, registers
`internal/product` routes explicitly (no reflection), and `framework.Run`.
Public product routes are Huma-typed under `/api/v1`. There is no generated
`service.go` or `repo.go` until `gombit make resource --service` / `--repo`.

`.env.example` lists `GOMBIT_*` server variables from the `config` package
and public `VITE_API_URL`. `VITE_*` is baked into the browser bundle — never
put secrets there. Access tokens stay in memory; generated source does not
use `localStorage`.

## `gombit dev`

From an application directory (the output of `gombit new`):

```sh
gombit dev
```

One command starts:

1. The Go API with reload when `air` or `watchexec` is on `PATH`. If neither
   is installed, Gombit runs `go run ./cmd/server` and prints a hint.
2. The Vite frontend with HMR (`pnpm` when available, otherwise `npm`). Vite
   proxies `/api`, `/openapi.json`, and `/docs` to the Go origin.
3. An OpenAPI watcher that regenerates `frontend/src/api/generated` when the
   live `/openapi.json` document changes (`gombit client generate`).

A service table is printed at startup:

```text
Backend      http://127.0.0.1:8080
Frontend     http://127.0.0.1:5173
OpenAPI      http://127.0.0.1:8080/openapi.json
API docs     http://127.0.0.1:8080/docs
```

### Flags

| Flag | Default | Role |
| --- | --- | --- |
| `--http` | `GOMBIT_HTTP_ADDR` or `:8080` | Go API listen address |
| `--frontend-host` | `127.0.0.1` | Vite bind host |
| `--frontend-port` | `5173` | Vite port |
| `--client-out` | `frontend/src/api/generated` | TypeScript client output |
| `--poll` | `1s` | OpenAPI poll interval |

`frontend/package.json` is required. A missing file is an error — backend-only
mode is not supported. Node.js is required for Vite.

`--http` and the Vite proxy origin are written into the child environment
(`GOMBIT_HTTP_ADDR`, `GOMBIT_DEV_BACKEND`, `VITE_API_URL=/api/v1`), replacing
any parent values so a shell-exported `.env.example` cannot keep the API on
`:8080` while the service table prints `--http :9090`.

SIGINT/SIGTERM stops the child processes. On Unix, Gombit signals the process
group so air/npm grandchildren exit. On Windows, teardown uses
`taskkill /T /F /PID` for the same process tree.

The scaffold's Vite stub is enough to start HMR. M5-1 replaces it with the
React skeleton.

## `gombit make resource`

From an application directory (the output of `gombit new`):

```sh
gombit make resource Widget name:string:required price:int
gombit make resource Invoice --service --repo --dry-run
gombit make resource Widget --force
```

`make` is a Cobra parent (`AddCommand`); `resource` is the subcommand. Root
help lists `make`.

This writes a feature-package under `internal/<snake>/`:

| File | When |
| --- | --- |
| `<snake>.go` | GORM model (`gorm.Model` + fields) |
| `handler.go` | Thin Huma list/get/create over GORM (D10 envelope) |
| `routes.go` | `Register(app *framework.App)` |
| `service.go` | Only with `--service` (pass-through) |
| `repo.go` | Only with `--repo` (pass-through) |

Default API prefix is `/api/v1`. The handler stays thin over GORM; `--service`
and `--repo` are C6 opt-in and are not used by the generated handler.

Route registration is appended in `cmd/server/main.go` via `go/ast` +
`go/parser` + `go/format` (never regex), next to `product.Register(app)`.
`internal/platform` AutoMigrate is updated the same way. Re-running does not
duplicate the `Register` call.

Frontend pages are vanilla TypeScript (list/table + create form) under
`frontend/src/<feature>/`. They import types from
`frontend/src/api/generated` — no hand-written API DTOs. A generated
`frontend/src/resources.ts` registry is the TypeScript registration point
(not regex-patched `main.ts`). Full React Router + React Hook Form is M5-1.

After generating routes, run `gombit client generate` or `gombit dev` so
`frontend/src/api/generated` exists.

### Field grammar

Design §27 subset:

```text
name:type[:required][,unique][,index]
```

Supported types: `string`, `text`, `int`, `int64`, `bool`, `uint`. Unknown
types error with the supported list. `nullable` is accepted as the opposite
of `required`.

### Idempotency

Generators print created/modified files. `--dry-run` writes nothing.
Identical re-runs are no-ops. A user-owned file (no generated banner) or a
generated file that differs from this run is refused unless `--force`.
`frontend/src/resources.ts` is the exception: it is always rewritten as a
scanned registry of generated feature pages (banner present), so edits to
that file are not preserved.

### Migrations

Gombit does not invent a migration DSL. The generated GORM model is
Atlas-loader ready. If the `atlas` binary is on `PATH`, `make resource`
attempts `migrations.MakeMigrations` with every `&pkg.Type{}` already
registered in `internal/platform` AutoMigrate plus the new model — Atlas
migrate diff treats that slice as the entire desired schema, so omitting
existing tables would emit DROP. If Atlas is missing from `PATH`, SQL is
skipped and the command prints:

```sh
gombit db makemigrations create_books \
  --model github.com/example/demo/internal/product.Product \
  --model github.com/example/demo/internal/book.Book
```

See [migrations.md](migrations.md).

## `gombit db`

Same subcommands and flags as M2, now on Cobra:

```sh
gombit db makemigrations create_products --model github.com/example/demo/internal/product.Product
gombit db migrate
gombit db rollback
gombit db status
gombit db seed
gombit db reset [--force]
```

See [migrations.md](migrations.md) for Atlas behavior.

## `gombit openapi` and `gombit client`

See [openapi.md](openapi.md) and [client.md](client.md).
