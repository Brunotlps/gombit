# Gombit

Gombit is a Django-for-Go full-stack framework. The current repository has
completed the M0 spike and is beginning M1 runtime work with typed config and
the initial `framework.App` lifecycle surface.

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
