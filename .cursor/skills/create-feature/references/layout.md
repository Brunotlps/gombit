# Generated application layout

Authoritative copy: `docs/GOMBIT_BUILD_PLAN.md` §3.2. Use this when scaffolding or placing application code — not for the framework repo itself until generators exist.

```
myapp/
├── cmd/server/main.go
├── internal/
│   ├── platform/            # app wiring the framework owns the shape of
│   └── product/             # one package per resource
│       ├── product.go       # model
│       ├── handler.go       # Huma handlers (thin, over GORM by default)
│       ├── service.go       # ONLY if --service
│       ├── repo.go          # ONLY if --repo
│       └── routes.go        # registration
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
- Route-registration edits (when generators exist) use `go/ast` and only append at a known registration point.
- Framework-owned endpoints (probes, metrics, OpenAPI) stay in the runtime; example-domain models must not leak into runtime packages.
- Split deploy is the default. Embed the frontend only via `gombit build --embed`.
- `.env.example` splits server secrets from `VITE_*` public values.
