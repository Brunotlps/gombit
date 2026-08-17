# Gombit CLI

`gombit` is a Cobra command tree (D13 / [ADR-014](adr/014-cli-cobra.md)).
Root help lists the command families. Nested families and app-registered
management commands attach with Cobra `AddCommand` (or `cli.AddCommand`,
a thin wrapper). There is no second command router and no reflection
discovery of commands.

The framework binary is `go run ./cmd/gombit` in this repository. Generated
apps get their own `cmd/gombit` that reuses `cli.NewRoot` and then calls
`product.RegisterCommands(root)` (and any `internal/commands.RegisterCommands`).
App-owned commands are discoverable via `go run ./cmd/gombit` from the
application module.

```sh
go run ./cmd/gombit --help
```

## Command families

| Family | Role | Milestone |
| --- | --- | --- |
| `gombit new` | Scaffold an application | M4-1 |
| `gombit dev` | Run API + Vite with HMR and live client regen | M4-2 |
| `gombit make resource` | Generate a feature-package resource (AST-safe) | M4-3 |
| `gombit make command` | Scaffold a Cobra management command (AST-safe) | M4-7 |
| `gombit db …` | Atlas-backed migrations | M2, migrated onto Cobra in M4-1 |
| `gombit openapi generate` | Write the live OpenAPI 3.1 document | M3-3 |
| `gombit client generate` / `check` | TypeScript client + drift | M3-4, M3-5 |
| `gombit routes` | Print HTTP routes | M4-4 |
| `gombit doctor` | Environment and config checks | M4-4 |
| `gombit config show` | Print typed config with secrets redacted | M4-4 |
| `gombit createsuperuser` | Create a superuser (admin) account | M4-6 |

## Generator golden tests

`goldentest` runs each generator against a fixed, non-interactive fixture,
diffs the output tree against `goldentest/testdata/golden`, compiles the
generated backend (`go build` with a local `replace` in a temp copy — never
committed), typechecks the frontend with `npx tsc --noEmit` when Node is
on `PATH` (`t.Skip` otherwise), and checks that a second run is idempotent
(`gombit new --force`, `make resource` without `--force`, `make command`
without `--force`, `client generate` without `--force`). Atlas is not
invoked, so migration filenames stay out of the trees.

```sh
go test ./goldentest
go test ./goldentest -update   # regenerate committed goldens
```

CI picks this up via `go test ./...`. Do not commit `replace` directives or
machine-specific paths in the goldens.

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
Vite + React + TypeScript skeleton (router, generated client, React Hook
Form). Bearer login is documented in [auth.md](auth.md). See [frontend.md](frontend.md).

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
├── cmd/gombit/main.go    # framework tree + product.RegisterCommands
├── internal/
│   ├── platform/
│   └── product/          # model, handler.go, routes.go, commands.go
├── database/migrations/
├── database/seeds/
├── config/
├── frontend/             # Vite + React skeleton (main.tsx, router, generated client)
├── gombit.yaml
├── .air.toml
├── .env.example
├── go.mod
└── README.md
```

`cmd/server/main.go` calls `config.Load()`, `framework.New`, registers
`internal/product` routes explicitly (no reflection), and `framework.Run`.
`cmd/gombit/main.go` calls `cli.NewRoot`, `product.RegisterCommands(root)`,
and `cli.ExecuteRoot`. Public product routes are Huma-typed under `/api/v1`.
There is no generated `service.go` or `repo.go` until
`gombit make resource --service` / `--repo`.

`.env.example` lists `GOMBIT_*` server variables from the `config` package
and public `VITE_API_URL` (empty means same-origin for the Vite `/api`
proxy). `VITE_*` is baked into the browser bundle — never put secrets there.
Access tokens stay in memory; generated source does not use `localStorage`.

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
(`GOMBIT_HTTP_ADDR`, `GOMBIT_DEV_BACKEND`, `VITE_API_URL=`), replacing
any parent values so a shell-exported `.env.example` cannot keep the API on
`:8080` while the service table prints `--http :9090`. Empty `VITE_API_URL`
is same-origin so OpenAPI paths such as `/api/v1/products` hit the Vite
proxy.

SIGINT/SIGTERM stops the child processes. On Unix, Gombit signals the process
group so air/npm grandchildren exit. On Windows, teardown uses
`taskkill /T /F /PID` for the same process tree.

The scaffold is a Vite + React + TypeScript skeleton. See
[frontend.md](frontend.md). Bearer login is documented in [auth.md](auth.md).

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

Frontend pages are React + TypeScript (list/table + React Hook Form create)
under `frontend/src/<feature>/`. They import types from
`frontend/src/api/generated` — no hand-written API DTOs — and map D10
`error.fields` through `frontend/src/api/formErrors.ts`. A generated
`frontend/src/resources.tsx` registry is the React Router registration
point (not regex-patched `main.tsx`).

After generating routes, run `gombit client generate` or `gombit dev` so
`frontend/src/api/generated` includes the new paths.

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
`frontend/src/resources.tsx` is the exception: it is always rewritten as a
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

## `gombit make command`

From an application directory (the output of `gombit new`):

```sh
gombit make command greet
gombit make command hello --package hello --dry-run
gombit make command greet --force
```

`make` is a Cobra parent; `command` is the subcommand. This writes a
feature-package management command and registers it on the **app-owned**
`cmd/gombit` tree — not the framework binary
(`go run github.com/LAA-Software-Engineering/gombit/cmd/gombit`).
The generator requires a `gombit new` app (`cmd/server/main.go` and/or
`gombit.yaml`) and refuses a framework-shaped tree so it cannot rewrite
the framework `cmd/gombit/main.go`.

Default package is `internal/commands`. `--package hello` writes
`internal/hello/` instead. Files:

| File | Role |
| --- | --- |
| `internal/<pkg>/<name>.go` | `New<Name>Command() *cli.Command` |
| `internal/<pkg>/commands.go` | `RegisterCommands(root *cli.Command)` calling `cli.AddCommand` |
| `cmd/gombit/main.go` | AST-appends `<pkg>.RegisterCommands(root)` next to `product.RegisterCommands(root)` |

`cli.AddCommand` is a thin wrapper around Cobra `AddCommand` (D13 /
[ADR-014](adr/014-cli-cobra.md)). Generated apps do not invent a second
router and do not discover commands by reflection.

After generating:

```sh
go run ./cmd/gombit greet
go run ./cmd/gombit --help
```

Registration edits use `go/ast` + `go/parser` + `go/format` (never regex).
`--dry-run` writes nothing. Re-running does not duplicate `RegisterCommands` or
`AddCommand` calls. A generated command file that differs from this run
(or a user-owned file) is refused unless `--force`. `commands.go` and
`cmd/gombit/main.go` are additive AST edits of known registration points.

Command names that collide with framework families (`new`, `dev`, `make`,
`db`, `openapi`, `client`, `routes`, `doctor`, `config`, `help`,
`completion`) are rejected.

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

## `gombit routes`

Print a table of HTTP method and path:

```sh
gombit routes
gombit routes --url http://127.0.0.1:8080
```

Default listing constructs an in-process `framework.App` (memory cache, docs
on) and prints framework-owned routes: `/livez`, `/readyz`, `/metrics`,
`/openapi.json` (and Huma siblings), and `/docs`. Application feature routes
are registered by the app; they are not discovered by reflection. Against a
running server, `--url` fetches `/openapi.json` and lists the live contract
paths (probes and other raw Gin routes stay out of that spec).

## `gombit doctor`

```sh
gombit doctor
gombit doctor --dir database/migrations
```

Prints a status table. A `FAIL` row exits non-zero.

| Check | Role |
| --- | --- |
| `go` | `go version` on `PATH` |
| `node` | `node --version`; **warn** if missing (`gombit dev` needs it) |
| `config` | `config.Load()` / `FieldErrors` |
| `database` | `database.Open` + ping when a driver/DSN is set (timeout-bounded) |
| `redis` | ping when `GOMBIT_CACHE_DRIVER=redis` (timeout-bounded) |
| `migrations` | pending Atlas SQL in `--dir` (**warn**); skip if the directory is absent |
| `http` | `GOMBIT_HTTP_ADDR` parses; **warn** if the port is in use |
| `insecure` | production `/docs`, unwritable SQLite path, production JWT secret shorter than 32 characters or equal to the generated-app development placeholder |

Network checks use a short timeout so CI cannot hang. Doctor does not start
Postgres/Redis containers; an invalid driver, broken `config.Load()`, or an
unwritable SQLite path is enough to flag a deliberately-broken config.

Appendix C rejects a production JWT secret shorter than 32 characters, and
the generated-app development placeholder, on the `config` row
(`config.Load`) and the `insecure` row when config is stubbed past Load. Cookie `Secure` and CORS+credentials checks wait for
M5-3. Production trusted-proxy and Redis `TLSInsecure` rejections already
live in `config.Validate()` and show up on the `config` row.

## `gombit config show`

```sh
gombit config show
```

Prints the typed `config.Load()` result as aligned `key<TAB>value` lines.
Database DSN userinfo/passwords, `Cache.Redis.Password`, and
`Auth.JWTSecret` are replaced with `*****` and must never appear in the
output.

See [config.md](config.md).

## `gombit createsuperuser`

```sh
gombit createsuperuser --email admin@example.com --password correct-horse-battery-staple
gombit createsuperuser --no-input --email admin@example.com --password correct-horse-battery-staple
gombit createsuperuser
```

Django `createsuperuser` → `gombit createsuperuser` (M4-6). It seeds an
admin account against the same `auth` users table as `POST /auth/register`,
using the same `auth.Service` bcrypt hasher — there is no second hash path.

Requires `GOMBIT_JWT_SECRET` to be set (`cfg.Auth.Enabled()`): without it
Bearer auth is unmounted by `framework.New`, so a superuser could never log
in. It loads config with `config.Load`, opens the database with
`database.Open`, runs `auth.Migrate` so the command also works against a
fresh database with no prior migrations, then calls
`auth.Service.CreateSuperuser`, which sets `User.IsSuperuser` and shares
`Register`'s unique-email path — duplicate emails are refused with the same
error as `/auth/register`.

### Flags

| Flag | Role |
| --- | --- |
| `--email` | Superuser account email |
| `--password` | Superuser account password (prefer the interactive prompt; visible in shell history and process listings when passed as a flag) |
| `--no-input` | Never prompt; require both `--email` and `--password` (use in scripts and tests) |

If `--email` or `--password` is omitted and stdin is a TTY, the command
prompts interactively (password entry is hidden). If stdin is not a TTY (CI,
tests, pipes), the missing flags are a hard error instead of a hang.
`--no-input` skips TTY detection entirely.

`IsSuperuser` is the only identity flag `auth.User` gains for this issue.
Groups/permissions are a product surface for the post-v0.1 admin milestone
(ADMIN-3), not part of M4-6.

## `gombit openapi` and `gombit client`

See [openapi.md](openapi.md) and [client.md](client.md).
