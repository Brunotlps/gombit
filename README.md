# Gombit

Gombit is a Django-for-Go full-stack framework. The current repository has
typed config, `framework.App` lifecycle, Huma contract/OpenAPI/client
generation, Atlas-backed `gombit db` migrations, and a Cobra CLI that
scaffolds apps with `gombit new`, runs them with `gombit dev`, and optionally
embeds the Vite frontend with `gombit build --embed`.

## Development

```sh
go test ./...
golangci-lint run
```

Emit the current spike OpenAPI document:

```sh
go run ./cmd/contract-spike-openapi openapi.json
```

Scaffold an application (Cobra CLI; default module `github.com/example/demo`):

```sh
go run ./cmd/gombit new demo --database sqlite
```

From the app directory, `gombit dev` runs the Go API and Vite frontend
together, proxies `/api` (and `/openapi.json`, `/docs`) to the backend,
regenerates the TypeScript client when the live spec changes, and prints a
service table including the API docs URL. See [`docs/cli.md`](docs/cli.md).

Write the live app spec (app must be serving `/openapi.json`):

```sh
go run ./cmd/gombit openapi generate --out openapi.json
```

Interactive docs are at `/docs` when `GOMBIT_DOCS_ENABLED` is on (the local
default). See [`docs/openapi.md`](docs/openapi.md).

The authoritative implementation backlog and architecture decisions live in
[`docs/GOMBIT_BUILD_PLAN.md`](docs/GOMBIT_BUILD_PLAN.md).

The Cobra command tree, `gombit new`, `gombit dev`, `gombit build --embed`,
`gombit make resource`,
`gombit make command`, `gombit routes`, `gombit doctor`, and
`gombit config show` are documented in [`docs/cli.md`](docs/cli.md).
Accepted architecture decisions are recorded under [`docs/adr/`](docs/adr/).
Runtime configuration is documented in [`docs/config.md`](docs/config.md).
Application lifecycle is documented in [`docs/lifecycle.md`](docs/lifecycle.md).
Application-owned route registration is documented in [`docs/router.md`](docs/router.md).
Huma DTO and validation conventions are documented in [`docs/contract.md`](docs/contract.md).
OpenAPI emission and `/docs` are documented in [`docs/openapi.md`](docs/openapi.md).
TypeScript client generation and the contract drift check are documented in
[`docs/client.md`](docs/client.md).
The Vite + React skeleton is documented in [`docs/frontend.md`](docs/frontend.md).
Optional `go:embed` production is documented in [`docs/build.md`](docs/build.md).
Bearer JWT login is documented in [`docs/auth.md`](docs/auth.md); cookie/session
auth (`--auth cookie`) and its CSRF threat model are documented in
[`docs/auth-cookie.md`](docs/auth-cookie.md).
The admin registry, introspection API, and framework-owned SPA under
`/admin/` (ADMIN-1 + ADMIN-2) are documented in
[`docs/admin.md`](docs/admin.md) ([ADR-013](docs/adr/013-runtime-generic-admin.md)).
Database runtime support is documented in [`docs/database.md`](docs/database.md).
Cache runtime support is documented in [`docs/cache.md`](docs/cache.md).
Runtime logging is documented in [`docs/logging.md`](docs/logging.md).
Migration generation is documented in [`docs/migrations.md`](docs/migrations.md).
