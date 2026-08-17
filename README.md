# Gombit

Gombit is a Django-for-Go full-stack framework. The current repository has
completed the M0 spike and is beginning M1 runtime work with typed config,
the `framework.App` lifecycle surface, and application-owned route
registration.

## Development

```sh
go test ./...
golangci-lint run
```

Emit the current spike OpenAPI document:

```sh
go run ./cmd/contract-spike-openapi openapi.json
```

Write the live app spec (app must be serving `/openapi.json`):

```sh
go run ./cmd/gombit openapi generate --out openapi.json
```

Interactive docs are at `/docs` when `GOMBIT_DOCS_ENABLED` is on (the local
default). See [`docs/openapi.md`](docs/openapi.md).

The authoritative implementation backlog and architecture decisions live in
[`docs/GOMBIT_BUILD_PLAN.md`](docs/GOMBIT_BUILD_PLAN.md).

Accepted architecture decisions are recorded under [`docs/adr/`](docs/adr/).
Runtime configuration is documented in [`docs/config.md`](docs/config.md).
Application lifecycle is documented in [`docs/lifecycle.md`](docs/lifecycle.md).
Application-owned route registration is documented in [`docs/router.md`](docs/router.md).
Huma DTO and validation conventions are documented in [`docs/contract.md`](docs/contract.md).
OpenAPI emission and `/docs` are documented in [`docs/openapi.md`](docs/openapi.md).
Database runtime support is documented in [`docs/database.md`](docs/database.md).
Cache runtime support is documented in [`docs/cache.md`](docs/cache.md).
Runtime logging is documented in [`docs/logging.md`](docs/logging.md).
Migration generation is documented in [`docs/migrations.md`](docs/migrations.md).
