# Contributing to Gombit

Gombit is built issue-by-issue from the backlog in
[`docs/GOMBIT_BUILD_PLAN.md`](docs/GOMBIT_BUILD_PLAN.md). Each pull request
should link the issue it satisfies and stay inside that issue's milestone.

## Local checks

Run the same baseline checks as CI before opening a pull request:

```sh
go test ./...
golangci-lint run
go run ./cmd/gombit --help
go run ./cmd/gombit client check
```

`gombit new demo --database sqlite` scaffolds a compiling app. See
[`docs/cli.md`](docs/cli.md).

`gombit client check --write` regenerates `examples/client/openapi.json` and
the sample TypeScript client. CI fails if those files would change.

## Working agreement

A pull request is not done unless it satisfies the Agent Working Agreement in
`docs/GOMBIT_BUILD_PLAN.md` section 5. In short:

- new behavior has tests;
- stable features ship docs and appear in an example app;
- extraction from existing templates preserves contracts;
- generators are idempotent, additive, and AST-safe for Go source edits;
- generated frontend source contains no secrets;
- API changes regenerate OpenAPI and the TypeScript client in the same PR;
- scope stays inside the issue milestone;
- the PR links its issue and states which acceptance criteria it satisfies.
