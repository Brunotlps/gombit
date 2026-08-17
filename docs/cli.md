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
| `gombit db …` | Atlas-backed migrations | M2, migrated onto Cobra in M4-1 |
| `gombit openapi generate` | Write the live OpenAPI 3.1 document | M3-3 |
| `gombit client generate` / `check` | TypeScript client + drift | M3-4, M3-5 |

Not in this milestone: `gombit make resource` (M4-3),
`routes` / `doctor` / `config show` (M4-4), `createsuperuser` (M4-6),
`make command` (M4-7).

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
├── frontend/             # Vite stub (package.json, vite.config.ts, src/main.ts)
├── gombit.yaml
├── .air.toml
├── .env.example
├── go.mod
└── README.md
```

`cmd/server/main.go` calls `config.Load()`, `framework.New`, registers
`internal/product` routes explicitly (no reflection), and `framework.Run`.
Public product routes are Huma-typed under `/api/v1`. There is no generated
`service.go` or `repo.go` (`--service` / `--repo` are M4-3).

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
