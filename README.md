# Gombit

Gombit is a Django-for-Go full-stack framework. The current repository has
typed config, `framework.App` lifecycle, Huma contract/OpenAPI/client
generation, Atlas-backed `gombit db` migrations, and a Cobra CLI that
scaffolds apps with `gombit new` and runs them with `gombit dev`.

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

The Cobra command tree, `gombit new`, `gombit dev`, `gombit routes`,
`gombit doctor`, and `gombit config show` are documented in
[`docs/cli.md`](docs/cli.md).
Accepted architecture decisions are recorded under [`docs/adr/`](docs/adr/).
Runtime configuration is documented in [`docs/config.md`](docs/config.md).
Application lifecycle is documented in [`docs/lifecycle.md`](docs/lifecycle.md).
Application-owned route registration is documented in [`docs/router.md`](docs/router.md).
Huma DTO and validation conventions are documented in [`docs/contract.md`](docs/contract.md).
OpenAPI emission and `/docs` are documented in [`docs/openapi.md`](docs/openapi.md).
TypeScript client generation and the contract drift check are documented in
[`docs/client.md`](docs/client.md).
Database runtime support is documented in [`docs/database.md`](docs/database.md).
Cache runtime support is documented in [`docs/cache.md`](docs/cache.md).
Runtime logging is documented in [`docs/logging.md`](docs/logging.md).
Migration generation is documented in [`docs/migrations.md`](docs/migrations.md).
