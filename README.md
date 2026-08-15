# Gombit

Gombit is a Django-for-Go full-stack framework. The current repository is in
the M0 spike stage and contains the module skeleton, project documentation,
CI wiring, and the M0-2 Huma-over-Gin contract spike.

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
