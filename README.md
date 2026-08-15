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

The authoritative implementation backlog and architecture decisions live in
[`docs/GOMBIT_BUILD_PLAN.md`](docs/GOMBIT_BUILD_PLAN.md).

Accepted architecture decisions are recorded under [`docs/adr/`](docs/adr/).
Runtime configuration is documented in [`docs/config.md`](docs/config.md).
Application lifecycle is documented in [`docs/lifecycle.md`](docs/lifecycle.md).
Application-owned route registration is documented in [`docs/router.md`](docs/router.md).
Database runtime support is documented in [`docs/database.md`](docs/database.md).
Runtime logging is documented in [`docs/logging.md`](docs/logging.md).
