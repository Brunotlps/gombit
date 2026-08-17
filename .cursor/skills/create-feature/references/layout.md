# Generated application layout

Authoritative copy: `docs/GOMBIT_BUILD_PLAN.md` §3.2. Use this when scaffolding or placing application code — not for the framework repo itself until generators exist.

```
myapp/
├── cmd/server/main.go
├── cmd/gombit/main.go       # framework Cobra tree + product.RegisterCommands
├── internal/
│   ├── platform/            # app wiring the framework owns the shape of
│   ├── commands/            # optional; gombit make command (M4-7)
│   └── product/             # one package per resource
│       ├── product.go       # model
│       ├── handler.go       # Huma handlers (thin, over GORM by default)
│       ├── service.go       # ONLY if --service
│       ├── repo.go          # ONLY if --repo
│       ├── routes.go        # HTTP registration
│       └── commands.go      # RegisterCommands for the app CLI
├── database/migrations/
├── database/seeds/
├── config/
├── frontend/                # Vite React app
├── gombit.yaml
├── .env.example
├── go.mod
└── README.md
```

Rules:

- `routes.go` is registered **explicitly** from `main.go`. No reflection discovery.
- Management commands are registered **explicitly** from `cmd/gombit` via Cobra `AddCommand` / `cli.AddCommand`. No reflection discovery and no second command router.
- Route-registration edits (when generators exist) use `go/ast` and only append at a known registration point.
- Framework-owned endpoints (probes, metrics, OpenAPI) stay in the runtime; example-domain models must not leak into runtime packages.
- Split deploy is the default. Embed the frontend only via `gombit build --embed`.
- `.env.example` splits server secrets from `VITE_*` public values.
