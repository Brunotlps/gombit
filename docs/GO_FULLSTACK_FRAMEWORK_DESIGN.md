# Go Full-Stack Web Framework — System Design Document

**Status:** Draft v0.1  
**Date:** 2026-08-14  
**Working name:** TBD (`fw` is used as a placeholder CLI name in examples)  
**Primary implementation language:** Go  
**Frontend:** React + TypeScript  
**Audience:** Framework maintainers, contributors, and advanced application developers

---

## 0. Executive Summary

This document defines the architecture and product direction for an opinionated, batteries-included full-stack web framework built around **Go + Gin + GORM** on the backend and **React + TypeScript + Vite** on the frontend.

The intended product category is not “a Go starter template.” The target is the developer experience and cohesion associated with **Django, Ruby on Rails, and Laravel**, while preserving the traits that make Go attractive: explicit code, simple deployment, compile-time checking, straightforward concurrency, and minimal runtime magic.

The framework will provide a blessed path for ordinary database-backed web applications:

- project creation and development orchestration;
- HTTP routing, controllers, middleware, validation, error handling, and lifecycle management;
- GORM-based persistence with **SQLite, PostgreSQL, and MySQL** as first-class database choices;
- versioned database migrations and seeders;
- authentication, authorization, sessions/JWT support, CSRF, and security defaults;
- caching, Redis integration, rate limiting, jobs, events, scheduling, mail, and storage over time;
- OpenAPI generation;
- generated TypeScript types and API clients from backend contracts;
- React + TypeScript application scaffolding with Vite;
- CRUD/resource generators spanning both backend and frontend;
- observability, health probes, logging, metrics, and tracing;
- optional gRPC without forcing protobuf or RPC concepts into ordinary web applications;
- a single CLI that makes common workflows discoverable and consistent;
- production builds that can optionally embed the frontend into the Go binary.

The project starts from two existing assets:

1. **`golang-rest-api-template`**, which already contains a substantial production-oriented Go backend foundation: Gin, GORM/PostgreSQL, JWT authentication, RBAC, Redis-backed rate limiting/cache, Swagger/OpenAPI, Prometheus metrics, OpenTelemetry tracing, request IDs/timeouts, security middleware, health probes, MongoDB access logging, Docker, unit tests, and E2E tests.
2. **`crud-template-monorepo`**, which already proves the full-stack composition of a Go/Gin backend with a React/TypeScript frontend, protected routes, CRUD screens, forms, Axios integration, and authentication.

The principal engineering task is therefore **extraction, generalization, convention design, code generation, and developer tooling** rather than greenfield implementation of every primitive.

---

# 1. Problem Statement

Go has excellent libraries for building web systems, but the normal workflow is compositional:

```text
router
+ ORM
+ validator
+ auth
+ migrations
+ cache
+ logging
+ metrics
+ tracing
+ frontend
+ API client
+ build tooling
+ application conventions
```

That flexibility is valuable, but it also means developers repeatedly make architectural and integration decisions that mature full-stack frameworks have already standardized.

The framework proposed here should answer:

> “How do I build a conventional database-backed web application in Go?”

with:

> “Use the blessed path, generate the project, write your domain logic, and deploy.”

The framework must be opinionated enough to eliminate routine decisions without becoming so magical that Go code stops feeling like Go.

---

# 2. Existing Baseline

## 2.1 `golang-rest-api-template`

The current backend template already includes:

- Gin HTTP server and routing;
- GORM with PostgreSQL;
- JWT login/register/refresh/logout;
- role-based authorization;
- API-key middleware;
- Redis cache;
- Redis-backed fixed-window rate limiting;
- Swagger/OpenAPI generation;
- Prometheus metrics;
- optional OpenTelemetry tracing;
- request IDs;
- request timeouts;
- maximum request-body limits;
- CORS and security middleware;
- liveness/readiness probes;
- Zap logging;
- MongoDB-backed request/access logging;
- Docker and Docker Compose;
- CI, unit tests, mocks, and E2E tests;
- graceful HTTP shutdown.

The current template still contains application-specific knowledge in framework-like code. For example:

- the router constructs `Book` and `User` handlers directly;
- route registration is hardcoded around `/books`, `/login`, `/register`, etc.;
- database startup is PostgreSQL-specific;
- `AutoMigrate` knows about `Book`, `User`, and `RefreshToken`;
- some configuration is read directly from environment variables in low-level packages.

Those seams are the first extraction targets.

## 2.2 `crud-template-monorepo`

The current monorepo proves the end-to-end application shape:

- Go/Gin REST backend;
- React + TypeScript frontend;
- JWT authentication;
- protected frontend routes;
- CRUD pages;
- Material UI;
- form validation;
- Axios API client;
- hand-written TypeScript API/domain types.

The frontend currently uses Create React App (`react-scripts`). The framework version should migrate to Vite and make the API contract generated rather than manually duplicated.

## 2.3 Existing Code That Should Be Preserved

The default strategy is **refactor and extract**, not rewrite.

The following existing investments should be retained where their contracts remain sound:

- JWT/RBAC implementation;
- request-ID and timeout middleware;
- rate limiting;
- security headers;
- metrics and tracing;
- health probes;
- Redis configuration logic;
- test coverage;
- graceful shutdown behavior;
- GORM repository implementations where reusable;
- Docker patterns;
- React CRUD behavior and frontend component patterns.

---

# 3. Product Definition

## 3.1 What This Project Is

A full-stack framework for building web applications with:

```text
Go
├── Gin
├── GORM
├── SQLite / PostgreSQL / MySQL
├── REST/JSON
├── optional gRPC
└── batteries-included infrastructure

React
├── TypeScript
├── Vite
├── generated API contracts
└── framework frontend helpers

CLI
├── project creation
├── development orchestration
├── generators
├── migrations
├── test/build commands
└── diagnostics
```

## 3.2 What This Project Is Not

It is not:

- a generic “choose every library yourself” toolkit;
- a thin wrapper around Gin;
- a thin wrapper around GORM;
- a Cookiecutter-only repository template;
- a microservice platform;
- a Kubernetes framework;
- an ORM replacement;
- a Java-style runtime dependency injection container;
- a frontend meta-framework intended to replace Next.js;
- a promise to abstract every database or persistence paradigm behind one interface.

---

# 4. Goals

## 4.1 Primary Goals

1. **Convention over configuration.** Provide a clear default architecture.
2. **Fast time to first useful application.** A new app should be productive within minutes.
3. **Full-stack type safety.** Backend contracts should generate frontend TypeScript types and clients.
4. **Database choice without application rewrite.** SQLite, PostgreSQL, and MySQL should be first-class.
5. **Excellent local development.** One command should run backend, frontend, and optional infrastructure.
6. **Production-quality defaults.** Security, observability, health checks, and graceful shutdown should not be afterthoughts.
7. **Go-native implementation style.** Prefer explicit constructors, interfaces at real boundaries, and compile-time behavior over reflection-heavy magic.
8. **Escape hatches.** Advanced users must be able to access Gin, GORM, `database/sql`, raw HTTP, and underlying clients.
9. **Incremental adoption.** Runtime packages should be usable independently where practical.
10. **Generated boilerplate must be understandable.** Code generation should create normal Go/TypeScript code that users can read and edit.

## 4.2 Secondary Goals

- optional single-binary deployment;
- optional gRPC transport;
- jobs/queues, scheduler, events, mail, storage;
- plugin/module ecosystem;
- admin scaffolding;
- optional UI presets;
- strong documentation and reference applications.

---

# 5. Non-Goals for v1

The first stable release does not need to support:

- MongoDB/DynamoDB/Cassandra as interchangeable primary databases;
- multiple ORMs;
- GraphQL;
- SSR as a core frontend requirement;
- runtime-loaded Go plugins;
- arbitrary frontend frameworks;
- distributed workflow orchestration;
- automatic cloud infrastructure provisioning;
- transparent switching between SQL and NoSQL semantics;
- zero-code application development.

These can be revisited after the core framework is mature.

---

# 6. Design Principles

## 6.1 Convention Over Configuration

The framework owns the boring decisions:

- Gin is the HTTP framework.
- GORM is the ORM.
- React + TypeScript is the frontend.
- Vite is the frontend build system.
- REST/JSON is the default browser/API transport.
- OpenAPI is the HTTP contract representation.
- application code follows a canonical directory structure.
- migrations, generators, auth, and resource conventions are framework-level concepts.

Users should configure application-specific behavior, not reconstruct the framework architecture.

## 6.2 Explicit Go, Minimal Magic

Do not imitate Ruby/PHP implementation techniques merely to imitate Rails/Laravel syntax.

Prefer:

```go
products := product.NewModule(app)
products.RegisterRoutes(app.Router())
```

over reflection-driven runtime discovery when explicit registration is simple.

Use code generation where it creates compile-time artifacts rather than runtime magic.

## 6.3 Interfaces at Boundaries, Not Everywhere

Good abstraction targets:

- cache;
- mail;
- queue;
- file/object storage;
- clock;
- external service clients;
- domain repositories where business logic benefits from substitution.

Poor abstraction target:

```go
type UniversalORM interface {
    Find(...)
    Where(...)
    Join(...)
    Preload(...)
    Transaction(...)
    ...
}
```

The framework will expose `*gorm.DB` rather than reimplement GORM through an interface.

## 6.4 Progressive Disclosure

The simplest application should not require understanding:

- Redis;
- OpenTelemetry;
- gRPC;
- distributed jobs;
- Docker;
- reverse proxy configuration.

Advanced features become visible as they are enabled.

## 6.5 Secure Defaults

Security-sensitive defaults should favor production-safe behavior, even when that requires explicit development overrides.

## 6.6 One Source of Truth for API Contracts

Do not maintain the same DTO shape manually in Go and TypeScript.

The framework contract pipeline is:

```text
Go HTTP contract
      ↓
   OpenAPI
      ↓
TypeScript types + client
      ↓
React application
```

---

# 7. Target Developer Experience

## 7.1 Create a Project

`fw` is a placeholder command name.

```bash
fw new bookstore
```

Interactive flow:

```text
Project name: bookstore
Database:
  > SQLite
    PostgreSQL
    MySQL

Frontend:
  > React + TypeScript

UI preset:
  > MUI
    Minimal
    None

Cache:
  > Memory
    Redis
    None

Authentication:
  > Session/cookie
    JWT bearer
    None

Generate example resource? [y/N]
```

Non-interactive:

```bash
fw new bookstore \
  --database postgres \
  --frontend react \
  --ui mui \
  --cache redis \
  --auth cookie
```

## 7.2 Run Development

```bash
fw dev
```

The command should:

1. validate configuration;
2. start required local infrastructure if configured and requested;
3. run migrations when policy allows;
4. start the Go server with reload support;
5. start Vite;
6. proxy frontend API requests to Go;
7. watch OpenAPI changes;
8. regenerate the TypeScript contract when necessary;
9. print a concise service table.

Example:

```text
Backend      http://localhost:8001
Frontend     http://localhost:5173
OpenAPI      http://localhost:8001/openapi.json
Swagger UI   http://localhost:8001/docs
Database     postgres://localhost:5432/bookstore
Redis        redis://localhost:6379
```

## 7.3 Generate a Resource

```bash
fw make resource Product \
  name:string:required \
  description:text \
  price:decimal:required \
  active:bool:default=true
```

This may generate:

```text
app/
├── models/product.go
├── services/product_service.go
├── controllers/product_controller.go
├── requests/product_request.go
├── policies/product_policy.go
└── modules/product_module.go

database/
└── migrations/20260814230000_create_products.go

frontend/src/
├── api/generated/...
├── pages/products/
│   ├── ProductListPage.tsx
│   ├── ProductCreatePage.tsx
│   ├── ProductEditPage.tsx
│   └── ProductDetailPage.tsx
└── features/products/
    ├── ProductForm.tsx
    └── ProductTable.tsx
```

The generator also registers routes using an AST-safe mechanism and regenerates the API contract.

## 7.4 Database Operations

```bash
fw db migrate
fw db rollback
fw db status
fw db seed
fw db reset
```

## 7.5 Introspection

```bash
fw routes
fw doctor
fw config show
fw openapi generate
fw client generate
fw version
```

---

# 8. High-Level Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                        Developer CLI                         │
│ new | dev | build | test | make | db | routes | doctor      │
└──────────────────────────┬───────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                      Application Project                     │
│                                                              │
│  ┌─────────────────────────┐   ┌──────────────────────────┐  │
│  │ Go application          │   │ React application        │  │
│  │                         │   │                          │  │
│  │ controllers             │   │ pages/features           │  │
│  │ services                │   │ forms/components         │  │
│  │ models                  │   │ generated API client     │  │
│  │ policies                │   │ generated TS types       │  │
│  │ jobs                    │   │ auth integration         │  │
│  └───────────┬─────────────┘   └─────────────┬────────────┘  │
│              │                               │               │
│              └──────── REST/JSON ────────────┘               │
└──────────────────────────┬───────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                     Framework Runtime                        │
│ Gin | GORM | auth | validation | cache | jobs | telemetry    │
│ config | lifecycle | migration | security | error handling   │
└──────────────┬───────────────────────────────┬───────────────┘
               │                               │
               ▼                               ▼
     ┌───────────────────┐             ┌────────────────────┐
     │ SQL database      │             │ Optional services  │
     │ SQLite/Postgres/  │             │ Redis, SMTP, S3,   │
     │ MySQL             │             │ OTLP, queues, etc. │
     └───────────────────┘             └────────────────────┘
```

Optional gRPC attaches to the same application/service layer:

```text
HTTP Controller ─┐
                 ├── Application Service ── Repository/GORM
gRPC Handler ────┘
```

---

# 9. Framework Repository Architecture

Initially, the framework should use a monorepo to coordinate Go runtime, CLI, frontend helpers, templates, and examples.

```text
framework/
├── go.mod
├── cmd/
│   └── fw/
├── runtime/
│   ├── app/
│   ├── config/
│   ├── http/
│   ├── database/
│   ├── migration/
│   ├── auth/
│   ├── authorization/
│   ├── validation/
│   ├── cache/
│   ├── jobs/
│   ├── events/
│   ├── scheduler/
│   ├── mail/
│   ├── storage/
│   ├── observability/
│   └── testing/
├── internal/
│   ├── generator/
│   ├── project/
│   └── templates/
├── frontend/
│   ├── package.json
│   └── src/
├── templates/
│   ├── app/
│   └── resource/
├── examples/
│   ├── minimal/
│   ├── bookstore/
│   └── multi-db/
└── docs/
```

Published artifacts may later be versioned independently if release coupling becomes painful.

---

# 10. Generated Application Structure

Recommended default:

```text
myapp/
├── cmd/
│   └── server/
│       └── main.go
├── app/
│   ├── controllers/
│   ├── models/
│   ├── services/
│   ├── repositories/
│   ├── requests/
│   ├── policies/
│   ├── middleware/
│   ├── jobs/
│   ├── events/
│   └── modules/
├── database/
│   ├── migrations/
│   ├── seeds/
│   └── factories/
├── routes/
│   ├── api.go
│   └── web.go
├── config/
├── frontend/
│   ├── src/
│   ├── public/
│   ├── vite.config.ts
│   └── package.json
├── tests/
├── framework.yaml
├── .env.example
├── Dockerfile
├── compose.yaml
├── go.mod
└── README.md
```

The structure is part of the framework contract. Users may deviate, but generators and documentation target this layout.

---

# 11. Application Lifecycle

## 11.1 Desired `main.go`

Application bootstrap should be intentionally small:

```go
package main

import (
    "log"

    "myapp/app"
    "github.com/example/framework"
)

func main() {
    if err := framework.Run(app.New()); err != nil {
        log.Fatal(err)
    }
}
```

## 11.2 Framework-Owned Lifecycle

The runtime owns:

1. config loading;
2. logging initialization;
3. database connection;
4. migration state checks;
5. optional Redis/cache connection;
6. auth infrastructure;
7. tracing and metrics;
8. HTTP server construction;
9. middleware installation;
10. module registration;
11. readiness/liveness endpoints;
12. frontend static asset mounting when embedded;
13. signal handling;
14. graceful shutdown;
15. dependency cleanup.

## 11.3 Lifecycle Hooks

Applications and packages may register hooks:

```go
app.OnStart(func(ctx context.Context) error {
    return nil
})

app.OnStop(func(ctx context.Context) error {
    return nil
})
```

Hooks run in deterministic order and have bounded shutdown contexts.

---

# 12. Configuration System

## 12.1 Precedence

Configuration resolution:

```text
framework defaults
    ↓
framework.yaml
    ↓
environment-specific config
    ↓
environment variables
    ↓
CLI flags
```

Secrets should not be required in committed YAML.

## 12.2 Example

```yaml
app:
  name: bookstore
  environment: development
  url: http://localhost:5173

server:
  host: 0.0.0.0
  port: 8001
  trusted_proxies: []

database:
  driver: postgres
  dsn_env: DATABASE_URL
  pool:
    max_open: 25
    max_idle: 5
    max_lifetime: 1h

cache:
  driver: redis
  url_env: REDIS_URL

auth:
  driver: cookie
  session_ttl: 24h

observability:
  metrics: true
  tracing: false

frontend:
  enabled: true
  dev_server: http://localhost:5173
  embed_on_build: true
```

## 12.3 Typed Config

Low-level framework packages should receive typed configuration instead of repeatedly reading `os.Getenv`.

Environment parsing belongs at the configuration boundary.

---

# 13. HTTP Layer

## 13.1 Gin Is the Native HTTP Runtime

The framework should not pretend Gin does not exist.

Expose controlled access:

```go
app.Router() *gin.Engine
```

or a framework router wrapper that still exposes the underlying Gin engine:

```go
app.HTTP().Gin()
```

## 13.2 Route Registration

Application routes should be explicit and discoverable.

```go
func Register(r *framework.Router, c *ProductController) {
    products := r.Group("/api/products")

    products.GET("", c.Index)
    products.GET("/:id", c.Show)
    products.POST("", c.Create)
    products.PUT("/:id", c.Update)
    products.DELETE("/:id", c.Delete)
}
```

A resource helper may reduce repetition:

```go
r.Resource("/api/products", controller)
```

but it must expand to documented routes and remain inspectable through `fw routes`.

## 13.3 Middleware Ordering

The framework defines a canonical global order:

1. panic recovery;
2. request ID;
3. tracing;
4. structured access logging;
5. proxy/IP resolution;
6. security headers;
7. CORS;
8. body-size limit;
9. request timeout;
10. rate limiting;
11. authentication context;
12. application middleware;
13. route handler.

Middleware order is a framework compatibility concern and should be tested.

## 13.4 Standard Response Envelope

Default successful API response:

```json
{
  "data": {},
  "meta": {}
}
```

Collections may include:

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 125
  }
}
```

Applications may opt out for specialized endpoints.

## 13.5 Standard Error Format

```json
{
  "error": {
    "code": "validation_error",
    "message": "The request contains invalid fields.",
    "fields": {
      "email": ["must be a valid email address"]
    },
    "request_id": "..."
  }
}
```

Stable machine-readable error codes are part of the frontend contract.

---

# 14. Validation and Request Binding

Validation must be a first-class framework feature.

Controllers should be able to bind and validate request DTOs consistently:

```go
type CreateProductRequest struct {
    Name  string          `json:"name" validate:"required,min=2,max=200"`
    Price decimal.Decimal `json:"price" validate:"required,gt=0"`
}

func (c *ProductController) Create(ctx *gin.Context) {
    req, ok := framework.Bind[CreateProductRequest](ctx)
    if !ok {
        return
    }

    product, err := c.service.Create(ctx, req)
    framework.Respond(ctx, product, err)
}
```

Validation failures feed the standard error envelope and generated OpenAPI schema.

---

# 15. Application Services and Repositories

## 15.1 Default Layering

The default dependency flow:

```text
HTTP/gRPC
   ↓
Controller/Transport
   ↓
Application Service
   ↓
Repository
   ↓
GORM / SQL
```

Business rules belong in services, not Gin handlers.

## 15.2 Avoid Per-Model Boilerplate When Possible

The framework can provide a generic CRUD repository:

```go
repo := repository.New[models.Product](app.DB())
```

Specialized repositories are introduced when custom persistence behavior is required.

Do not generate an interface and implementation pair for every trivial model unless the application explicitly asks for one.

## 15.3 GORM Escape Hatch

Applications can always use:

```go
db := app.DB()
```

and receive `*gorm.DB`.

Raw SQL remains available through:

```go
sqlDB, err := app.DB().DB()
```

---

# 16. Database Architecture

## 16.1 Canonical ORM

**GORM is the supported ORM for v1.**

The framework does not attempt to abstract GORM away.

## 16.2 First-Class SQL Drivers

Required:

- SQLite;
- PostgreSQL;
- MySQL.

Potential later support:

- SQL Server;
- other GORM-supported relational databases where demand justifies a conformance target.

## 16.3 Driver Selection

```yaml
database:
  driver: sqlite
  dsn: ./storage/app.db
```

```yaml
database:
  driver: postgres
  dsn_env: DATABASE_URL
```

```yaml
database:
  driver: mysql
  dsn_env: DATABASE_URL
```

Internally:

```go
func Open(cfg Config) (*gorm.DB, error) {
    switch cfg.Driver {
    case SQLite:
        return gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{})
    case Postgres:
        return gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
    case MySQL:
        return gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
    default:
        return nil, ErrUnsupportedDriver
    }
}
```

## 16.4 Capability Awareness

The framework must not pretend SQL dialects are behaviorally identical.

Expose:

```go
app.Database().Driver()
app.Database().Capabilities()
```

Example capability flags:

- transactional DDL;
- returning clause;
- JSON column support;
- native UUID support;
- partial indexes;
- generated columns;
- full-text search behavior.

Generators should target portable defaults unless the user explicitly requests a dialect-specific feature.

## 16.5 Connection Pooling

Pool configuration is driver-aware and typed.

SQLite may intentionally use different pooling defaults than client/server databases.

---

# 17. Migrations

## 17.1 Versioned Migrations Are Required

`AutoMigrate` is useful during prototyping, but production schema evolution requires an ordered history.

The framework therefore has a migration subsystem.

```bash
fw make migration create_products
fw db migrate
fw db rollback
fw db status
```

## 17.2 Migration Representation

Preferred v1 direction: Go migration files using a small framework schema API backed by GORM's migrator primitives.

```go
type CreateProducts struct{}

func (CreateProducts) Up(m migration.Schema) error {
    return m.CreateTable("products", func(t *migration.Table) {
        t.ID()
        t.String("name", 200).NotNull()
        t.Decimal("price", 12, 2).NotNull()
        t.Boolean("active").Default(true)
        t.Timestamps()
    })
}

func (CreateProducts) Down(m migration.Schema) error {
    return m.DropTable("products")
}
```

The schema API maps portable operations to each supported dialect.

## 17.3 Dialect-Specific Escape Hatch

```go
if m.Driver() == database.Postgres {
    return m.Exec(`CREATE INDEX ...`)
}
```

Portability is the default, not a prison.

## 17.4 Migration Metadata

Use a framework-owned migrations table:

```text
framework_migrations
- version
- name
- batch
- applied_at
- checksum
```

Checksums detect accidental mutation of already-applied migrations.

## 17.5 CI Matrix

Every migration and database conformance test must run against:

```text
SQLite
PostgreSQL
MySQL
```

This is mandatory to claim official support.

---

# 18. Models, IDs, and Common Fields

Provide optional base types rather than mandatory inheritance.

```go
type Model struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

Alternative UUID model:

```go
type UUIDModel struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

Because UUID storage differs by dialect, UUID helpers must use driver-aware types or portable encodings.

---

# 19. Cache and Redis

## 19.1 Cache Is a Real Abstraction Boundary

Unlike the ORM, cache semantics are narrow enough to abstract cleanly.

The current cache interface leaks `go-redis` command types. Framework v1 should normalize it:

```go
type Cache interface {
    Get(ctx context.Context, key string, dst any) (bool, error)
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    Increment(ctx context.Context, key string, delta int64) (int64, error)
}
```

Optional advanced Redis access:

```go
app.Redis() *redis.Client
```

when Redis is enabled.

## 19.2 Drivers

- memory;
- Redis;
- no-op/disabled.

Memory cache is ideal for SQLite/local development.

## 19.3 Namespacing

Framework cache keys should be namespaced by application and environment.

---

# 20. Authentication

## 20.1 Authentication Must Support Multiple Application Shapes

Required v1 modes:

1. browser session / secure cookie;
2. JWT bearer API authentication;
3. authentication disabled.

## 20.2 Browser Default

For a first-party React web application, the preferred default should be **Secure, HttpOnly, SameSite cookies** rather than persisting bearer tokens in `localStorage`.

Reasons:

- JavaScript cannot read HttpOnly cookies;
- XSS exposure of long-lived tokens is reduced;
- browser authentication becomes an integrated framework concern.

Cookie auth requires CSRF protection for state-changing requests.

## 20.3 JWT Mode

Bearer JWT remains important for:

- mobile applications;
- external API consumers;
- service clients;
- stateless API deployments.

Refresh-token rotation and denylisting may use database and/or Redis implementations.

## 20.4 API Keys

API keys are suitable for server-to-server or external client identification.

A secret API key **must not be treated as secret when shipped inside browser JavaScript**. The generated React application should not embed a privileged application API key.

## 20.5 Password Storage

Password hashing is framework-owned and configurable behind a stable hasher interface.

Existing bcrypt behavior can be retained initially; Argon2id can be evaluated as an additional/default scheme with transparent rehashing policy.

---

# 21. Authorization

Provide:

- roles;
- permissions;
- policy objects;
- route guards;
- resource-level checks.

Example:

```go
type ProductPolicy struct{}

func (ProductPolicy) Update(user auth.User, product Product) bool {
    return user.ID == product.OwnerID || user.HasRole("admin")
}
```

Controllers should call a consistent authorization API:

```go
if !framework.Authorize(ctx, productPolicy.Update, product) {
    return
}
```

RBAC remains available as a simple fast path.

---

# 22. Security Baseline

Framework-owned defaults:

- secure headers;
- CORS policy;
- CSRF for cookie-authenticated browser requests;
- request size limits;
- request timeouts;
- rate limits;
- trusted-proxy configuration;
- secure cookie attributes;
- password hashing;
- secret validation;
- no debug stack traces in production;
- structured security logging;
- dependency vulnerability checks in CI;
- safe file upload defaults;
- protection against obvious path traversal;
- content-type validation;
- optional CSP configuration.

Security middleware should be composable, but disabling important protections in production should produce visible warnings.

---

# 23. OpenAPI and Backend-to-Frontend Contracts

## 23.1 Contract Pipeline

```text
Go request/response DTOs + route metadata
                 ↓
               OpenAPI
                 ↓
      TypeScript schema/types
                 ↓
         generated API client
                 ↓
          React hooks/services
```

OpenAPI is an intermediate representation and public artifact.

## 23.2 Generated TypeScript

Generated output must not be hand-edited:

```text
frontend/src/api/generated/
```

Example:

```ts
export interface Product {
  id: number;
  name: string;
  price: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateProductRequest {
  name: string;
  price: string;
  active?: boolean;
}
```

## 23.3 Generated Client

```ts
const product = await api.products.get({ id });
await api.products.create({ body: input });
```

The client should expose typed error objects that map to the framework error envelope.

## 23.4 Contract Stability

Breaking API changes should be detectable in CI through OpenAPI diffing once the project reaches stable release maturity.

---

# 24. React Frontend Architecture

## 24.1 Core Stack

- React;
- TypeScript;
- Vite;
- React Router;
- generated API client;
- framework auth helpers;
- framework error/validation helpers.

## 24.2 UI Libraries

A component library should not be part of the framework runtime contract.

Recommended starter presets:

- MUI default, because the existing monorepo already uses it successfully;
- minimal/no UI preset;
- additional presets later.

This avoids making the core framework dependent on the lifecycle of one design system.

## 24.3 Application Layout

```text
frontend/src/
├── app/
│   ├── router.tsx
│   └── providers.tsx
├── api/
│   └── generated/
├── components/
├── features/
├── hooks/
├── layouts/
├── pages/
├── styles/
└── main.tsx
```

## 24.4 Forms

The default starter can use React Hook Form.

Backend validation remains authoritative. Frontend form errors should map server field errors into the form automatically.

## 24.5 Data Fetching

Do not hardcode an irreversible data-fetching library into framework contracts in v1.

The generated client should be framework-neutral enough to use directly or wrap with TanStack Query later.

---

# 25. Development Server Integration

## 25.1 Development Mode

```text
Browser
   ↓
Vite :5173
   ├── frontend assets + HMR
   └── /api proxy
          ↓
       Gin :8001
```

This largely eliminates CORS complexity in the default local setup.

## 25.2 Production Mode — Single Binary

Optional default deployment:

```text
Vite build
   ↓
frontend/dist
   ↓
go:embed
   ↓
single Go executable
```

Gin serves API routes and static assets. SPA fallback serves `index.html` for frontend routes.

## 25.3 Production Mode — Split Deployment

Also support:

```text
Static frontend/CDN
        ↓
      browser
        ↓
Go API service
```

The framework must not require single-binary deployment.

---

# 26. CLI Architecture

The CLI is a first-class product component.

**Implementation library:** Cobra (`spf13/cobra`) — locked as build plan D13 /
ADR-014. Prefer Cobra's command tree and `AddCommand` for nested families and
for Django-style app-registered management commands (M4-7). Do not adopt Kong
or grow the pre-M4 stdlib `flag` router into the long-term architecture.

## 26.1 Command Families

```text
fw new
fw dev
fw build
fw test

fw make model
fw make request
fw make service
fw make controller
fw make policy
fw make migration
fw make resource
fw make job
fw make event
fw make middleware

fw db migrate
fw db rollback
fw db status
fw db seed
fw db reset

fw routes
fw config show
fw doctor

fw openapi generate
fw client generate
```

Aliases may provide Rails/Laravel-style syntax later.

## 26.2 `fw doctor`

Checks:

- Go version;
- Node/package manager;
- config validity;
- database connectivity;
- Redis connectivity;
- migration status;
- writable directories;
- frontend dependencies;
- OpenAPI generation;
- port conflicts;
- production-insecure settings.

## 26.3 Generator Implementation

Rules:

- use Go's `go/ast`, `go/parser`, and `go/format` for modifying Go source;
- never patch Go source with fragile regular expressions;
- prefer generated standalone TypeScript files over mutating arbitrary user code;
- when TypeScript modification is necessary, use predictable registry files or an AST tool;
- generators must be idempotent when practical;
- never silently overwrite edited application files;
- support `--dry-run` and `--force`;
- print created/modified files.

---

# 27. Resource Generator Schema

A compact field grammar is useful:

```bash
fw make resource Invoice \
  number:string:required,unique \
  customer_id:uint:required,index \
  amount:decimal:required \
  due_at:datetime \
  paid:bool:default=false
```

Initial scalar vocabulary:

```text
string
text
bool
int
uint
float
decimal
date
datetime
uuid
json
```

Modifiers:

```text
required
nullable
unique
index
default=...
min=...
max=...
```

Relations can be introduced explicitly:

```text
customer_id:uint:references=customers.id
```

The grammar must be versioned because it becomes part of the CLI API.

---

# 28. Jobs and Queues

Not required for the earliest alpha, but part of the framework direction.

Interface:

```go
type Queue interface {
    Push(ctx context.Context, job Job) error
}
```

Initial drivers:

- sync;
- database;
- Redis.

Generated jobs:

```bash
fw make job SendWelcomeEmail
```

Jobs need:

- retries;
- backoff;
- timeout;
- idempotency guidance;
- dead-letter/failure storage;
- observability;
- graceful worker shutdown.

Workers can run as a separate command:

```bash
fw worker
```

or separate Go binary generated from the same application.

---

# 29. Events

Application-local events decouple modules:

```go
events.Publish(ctx, UserRegistered{UserID: user.ID})
```

Listeners may be synchronous or queued.

Events are not a replacement for distributed messaging semantics unless an explicit driver provides them.

---

# 30. Scheduler

Provide a framework scheduler:

```go
scheduler.Daily("cleanup-sessions", cleanupExpiredSessions)
scheduler.Every("sync-stats", 15*time.Minute, syncStats)
```

CLI:

```bash
fw scheduler run
```

Distributed locking is required when multiple scheduler replicas are possible.

---

# 31. Mail

Stable interface:

```go
type Mailer interface {
    Send(ctx context.Context, message Message) error
}
```

Initial drivers:

- SMTP;
- log/console for development.

Templates can be Go templates or React/email integration later, but email rendering should not block core framework v1.

---

# 32. File and Object Storage

Stable interface:

```go
type Storage interface {
    Put(ctx context.Context, key string, r io.Reader, opts PutOptions) error
    Open(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    URL(ctx context.Context, key string, opts URLOptions) (string, error)
}
```

Initial drivers:

- local filesystem;
- S3-compatible storage later.

---

# 33. Optional gRPC

## 33.1 Positioning

gRPC is optional and should not affect a normal web application's mental model.

Enable:

```bash
fw add grpc
```

or project creation:

```bash
fw new service --grpc
```

## 33.2 Architecture

```text
REST controller ─┐
                 ├── service ── repository
gRPC handler ────┘
```

Business services should not depend on Gin or protobuf-generated types.

## 33.3 Contract Strategy

Do not automatically turn GORM models into protobuf contracts.

gRPC should remain proto-first for RPC boundaries, with mapping into application DTOs/services.

This preserves gRPC's explicit contract model and avoids coupling persistence schema to wire schema.

## 33.4 Gateway

HTTP/gRPC gateway support may be added for service-oriented deployments, but it is separate from the default React REST flow.

---

# 34. Logging

Structured logging is framework-owned.

Zap is a suitable default based on the existing backend.

Log records should include:

- timestamp;
- level;
- message;
- environment;
- service/application;
- request ID;
- trace/span IDs when available;
- route;
- method;
- status;
- latency;
- authenticated user ID when appropriate.

MongoDB should become an **optional log sink**, not a mandatory runtime dependency.

Default production sink: stdout/stderr structured logs.

---

# 35. Metrics, Tracing, and Health

Preserve the existing strong observability baseline.

## 35.1 Metrics

Prometheus metrics:

- request counts;
- status code counts;
- latency histograms;
- active requests;
- DB pool statistics;
- queue statistics;
- cache hit/miss counters where appropriate.

## 35.2 Tracing

OpenTelemetry is optional but first-class.

Trace context should flow:

```text
HTTP/gRPC
  ↓
service
  ↓
database/cache/external request
```

## 35.3 Health

- `/livez`: process liveness only;
- `/readyz`: dependency readiness;
- `/metrics`: configurable exposure;
- framework diagnostic endpoint only in safe environments.

Readiness dependency policy is configurable to avoid unnecessary cascading failure.

---

# 36. Testing Strategy

## 36.1 Framework Unit Tests

Test each runtime package independently.

## 36.2 Database Conformance Tests

Run the same suite against:

- SQLite;
- PostgreSQL;
- MySQL.

Test:

- CRUD;
- transactions;
- migration up/down;
- timestamps;
- nullable values;
- indexes;
- unique constraints;
- decimal handling;
- JSON behavior where portable;
- pagination;
- generated schema.

## 36.3 Generator Golden Tests

For every generator:

1. run against a fixture project;
2. compare generated tree and source to golden files;
3. compile backend;
4. typecheck frontend;
5. run formatting;
6. verify regeneration/idempotency behavior.

## 36.4 Contract Tests

Generate OpenAPI and TypeScript client, then compile the frontend against it.

## 36.5 End-to-End Tests

Reference app E2E flow:

```text
create app
migrate
register/login
create resource
list resource
update resource
delete resource
logout
```

Run against each officially supported database where feasible.

## 36.6 Security Tests

Test:

- auth bypass attempts;
- role checks;
- CSRF;
- CORS;
- rate limiting;
- token refresh/revocation;
- cookie attributes;
- request size limits;
- trusted proxies;
- common header protections.

---

# 37. Build and Packaging

## 37.1 `fw build`

Expected steps:

1. validate configuration;
2. generate OpenAPI;
3. generate TypeScript client;
4. typecheck frontend;
5. build frontend;
6. run Go generation if required;
7. embed frontend when configured;
8. compile Go binary;
9. print artifact path and build metadata.

## 37.2 Reproducibility

Build metadata:

```text
framework version
application version
git commit
build time
Go version
frontend build version
```

Support deterministic/reproducible builds where practical.

## 37.3 Containers

Generate a multi-stage Dockerfile:

```text
Node build stage
      ↓
Go build stage
      ↓
minimal runtime image
```

SQLite deployments need persistent volume guidance.

---

# 38. Local Infrastructure

Project creation should make the easy case genuinely easy.

## 38.1 SQLite

```bash
fw new notes --database sqlite
fw dev
```

No Docker required.

## 38.2 PostgreSQL/MySQL

The generated `compose.yaml` can include the selected DB.

Redis is included only when enabled.

## 38.3 Compose Profiles

Optional services can use Compose profiles rather than a proliferation of unrelated compose files.

---

# 39. Module and Extension Model

Avoid runtime Go plugin loading.

Framework modules are ordinary compile-time Go packages:

```go
app.Use(mail.Module(...))
app.Use(queue.Module(...))
app.Use(storage.S3(...))
```

A module may register:

- config;
- lifecycle hooks;
- routes;
- middleware;
- commands;
- migration sources;
- services.

Third-party modules are imported through Go modules, preserving normal Go dependency management.

Frontend companion packages are normal npm packages.

---

# 40. Dependency Injection

Avoid a Java-style reflection container.

Use explicit typed wiring.

Example module:

```go
func NewProductModule(app *framework.App) *ProductModule {
    repo := repository.New[Product](app.DB())
    service := NewProductService(repo)
    controller := NewProductController(service)

    return &ProductModule{
        Service: service,
        Controller: controller,
    }
}
```

A small service registry may exist for cross-module integration, but typed constructors remain the default.

Generated code can eliminate repetitive wiring.

---

# 41. Error Architecture

Define framework error categories:

```text
validation
authentication
authorization
not_found
conflict
rate_limited
dependency_unavailable
internal
```

Application services return domain/framework errors, not Gin responses.

Transport adapters map errors to HTTP/gRPC status codes.

This allows optional gRPC without contaminating services with HTTP semantics.

---

# 42. Pagination, Filtering, and Sorting

Resource endpoints should share conventions.

Example:

```text
GET /api/products?page=2&per_page=25&sort=-created_at&filter[active]=true
```

Default limits prevent unbounded queries.

Generators can produce a safe allowed-field list so arbitrary query parameters cannot become arbitrary SQL.

---

# 43. Transactions

Provide a simple transaction helper:

```go
err := app.Database().Transaction(ctx, func(tx *gorm.DB) error {
    // application operations
    return nil
})
```

Service/repository patterns must permit passing transaction-bound DB handles without global mutable state.

---

# 44. Multi-Tenancy

Not a v1 core feature.

The architecture should avoid blocking future strategies:

- tenant column;
- schema-per-tenant;
- database-per-tenant.

Do not prematurely add tenant context to every API.

---

# 45. Internationalization

Backend error codes should be stable and machine-readable.

Human-readable API messages should not become the only localization mechanism.

Frontend i18n can be an optional package/preset later.

---

# 46. Documentation Strategy

Documentation is a core product surface.

Required documentation sets:

1. tutorial: build a complete CRUD app;
2. concepts: lifecycle, modules, config, auth, database, frontend contract;
3. CLI reference;
4. API/runtime reference;
5. recipes;
6. deployment guide;
7. security guide;
8. migration guide from raw Gin/GORM projects;
9. contributor architecture guide;
10. upgrade guides.

Every stable framework feature needs:

- docs;
- tests;
- example;
- CLI discoverability where relevant.

---

# 47. Versioning and Compatibility

Use Semantic Versioning.

Before `v1.0`, breaking changes are allowed but must be documented.

After `v1.0`:

- public runtime packages have compatibility expectations;
- CLI behavior is a public API;
- project structure expected by generators is a public API;
- generator field grammar is versioned;
- generated code contains framework version metadata;
- deprecations should survive at least one minor release where practical.

The CLI should detect framework/application version mismatches.

---

# 48. Upgrade Strategy

Future command:

```bash
fw upgrade
```

The tool may:

- update Go module dependency;
- update frontend package dependency;
- apply safe template migrations;
- flag manual changes;
- run a compatibility doctor.

Do not blindly rewrite user code.

Codemods must be explicit, reviewable, and ideally produce diffs.

---

# 49. Performance Principles

The framework should add minimal overhead over Gin/GORM.

Rules:

- no request-time reflection-heavy dependency graph resolution;
- avoid excessive middleware allocations;
- lazy initialize optional subsystems where sensible;
- keep generated code direct;
- expose profiling hooks;
- benchmark common paths;
- avoid hidden N+1 queries in generated resources;
- paginate list endpoints;
- never use Redis `KEYS` in production paths.

Performance claims should be benchmarked, not marketed from intuition.

---

# 50. Security and Secret Management in the Frontend

The existing CRUD template demonstrates a critical issue the framework should correct: a browser bundle cannot protect an embedded API secret.

Rules:

- `VITE_*` values are public browser configuration;
- secrets never belong in generated frontend source;
- API keys intended as secrets are server-to-server only;
- cookie-based browser auth uses HttpOnly cookies;
- bearer tokens used by browser mode should have a clearly documented threat model;
- generated `.env.example` separates server secrets from public frontend values.

---

# 51. Recommended Initial Technology Decisions

| Area | Decision |
|---|---|
| Backend language | Go |
| HTTP | Gin |
| ORM | GORM |
| Databases | SQLite, PostgreSQL, MySQL |
| Default transport | REST/JSON |
| Optional transport | gRPC |
| API schema | OpenAPI |
| Frontend | React + TypeScript |
| Frontend build | Vite |
| Routing | React Router |
| Default UI preset | MUI (proposed) |
| Cache | memory / Redis / disabled |
| Auth | secure cookie and JWT bearer |
| Logging | Zap structured logging |
| Metrics | Prometheus |
| Tracing | OpenTelemetry |
| CLI implementation | Go |
| Deployment | split or optional single Go binary |
| Framework extension | compile-time Go modules |

---

# 52. Phased Implementation Plan

## Phase 0 — Foundation Extraction

Goal: turn the existing REST template into application-agnostic runtime packages.

Tasks:

- introduce typed config;
- extract `framework.App`;
- move lifecycle ownership into runtime;
- remove `Book`/`User` knowledge from router/bootstrap;
- make route registration application-owned;
- make migration/model registration application-owned;
- normalize cache interface;
- make Mongo logging optional;
- preserve tests and observability;
- establish framework repository layout.

Exit criteria:

- minimal example app boots through the framework runtime;
- runtime package contains no example-domain models;
- existing backend security/observability tests still pass.

## Phase 1 — Multi-Database Runtime

Goal: official SQLite/PostgreSQL/MySQL support.

Tasks:

- database driver registry;
- typed DSN/pool config;
- migration subsystem;
- driver conformance suite;
- generated Compose config;
- SQLite zero-infrastructure path.

Exit criteria:

- same example CRUD app passes E2E tests on all three DBs.

## Phase 2 — React/Vite Integration

Goal: first coherent full-stack alpha.

Tasks:

- migrate existing frontend patterns to Vite;
- create frontend skeleton;
- development proxy;
- generated OpenAPI artifact;
- generated TS types/client;
- auth integration;
- standard API error mapping;
- optional frontend embedding.

Exit criteria:

```bash
fw new demo --database sqlite
fw dev
```

opens a working authenticated application.

## Phase 3 — CLI and Generators

Goal: framework-level productivity.

Tasks:

- `fw new`;
- `fw make model`;
- `fw make migration`;
- `fw make resource`;
- `fw routes`;
- `fw doctor`;
- AST-safe code modification;
- generator golden tests.

Exit criteria:

A generated resource works backend-to-frontend without manual contract duplication.

## Phase 4 — Production Application Services

Goal: reach “batteries included.”

Tasks:

- queue/jobs;
- events;
- scheduler;
- mail;
- storage;
- richer authorization policies;
- production deployment docs;
- upgrade tooling foundation.

## Phase 5 — Optional gRPC

Goal: service-oriented extension without destabilizing the web core.

Tasks:

- gRPC module;
- proto/project layout;
- transport adapters;
- service-layer examples;
- tracing/auth integration;
- optional gateway support.

## Phase 6 — v1 Stabilization

Tasks:

- API compatibility review;
- security audit;
- performance benchmark suite;
- documentation completion;
- migration/upgrade guarantees;
- conformance matrix;
- reference applications;
- release policy.

---

# 53. Suggested Immediate Refactor of `golang-rest-api-template`

## 53.1 Router

Current:

```text
NewRouter(...)
  ├── constructs Book handler
  ├── constructs User handler
  ├── registers Book routes
  └── registers auth routes
```

Target:

```text
framework.New(...)
  ├── constructs generic Gin engine
  ├── installs framework middleware
  ├── mounts framework endpoints
  └── calls application/module route registration
```

Application:

```go
func RegisterRoutes(app *framework.App) {
    productModule := product.NewModule(app)
    productModule.RegisterRoutes(app.Router())

    authModule := account.NewModule(app)
    authModule.RegisterRoutes(app.Router())
}
```

## 53.2 Database

Current database bootstrap is PostgreSQL-specific and directly migrates application models.

Target:

```go
db, err := database.Open(cfg.Database)
```

Then migrations are discovered/registered by the application.

## 53.3 Environment Access

Replace scattered `os.Getenv` calls with typed config injection.

## 53.4 Cache

Replace Redis-command-returning cache interface with backend-neutral value semantics.

## 53.5 Logging

Keep Zap; make Mongo a selectable sink/module.

---

# 54. Suggested Immediate Refactor of `crud-template-monorepo`

1. replace Create React App with Vite;
2. remove hardcoded API URL;
3. remove browser-embedded API secret;
4. replace manual TypeScript DTO duplication with generated contracts;
5. centralize auth provider;
6. use standard API error mapping;
7. organize CRUD screens by feature/resource;
8. ensure frontend is generated from the same canonical resource conventions used by CLI;
9. make MUI a preset rather than core runtime dependency;
10. support frontend embedding during production build.

---

# 55. Key Architectural Risks

## 55.1 Scope Explosion

The framework can easily become “rewrite Laravel in Go.”

Mitigation:

- strict phased roadmap;
- stable core before broad feature count;
- one excellent implementation per subsystem before alternative drivers.

## 55.2 Database Portability Illusion

SQLite/Postgres/MySQL differ.

Mitigation:

- capability model;
- portable generator defaults;
- driver-specific escape hatch;
- mandatory multi-DB CI.

## 55.3 Generator Fragility

Generators that edit arbitrary source can break projects.

Mitigation:

- Go AST tooling;
- registry files;
- golden tests;
- dry-run;
- no silent overwrite.

## 55.4 Excessive Abstraction

Trying to hide Gin/GORM can create a second inferior framework inside the framework.

Mitigation:

- expose underlying primitives;
- abstract narrow boundaries only.

## 55.5 Frontend/Backend Version Skew

Generated client and server contract may diverge.

Mitigation:

- generation integrated into build/test;
- generated artifacts carry schema/version metadata;
- CI verifies clean regeneration.

## 55.6 Security Footguns

Convenience features can create dangerous defaults.

Mitigation:

- secure cookie defaults;
- no browser secrets;
- production configuration validation;
- security-focused conformance tests.

## 55.7 Framework Upgrade Pain

Generated application code can drift.

Mitigation:

- generated code should be ordinary and lightly coupled;
- runtime features preferred over continuously rewriting application code;
- codemods are explicit and versioned.

---

# 56. Architectural Decision Records — Initial Set

## ADR-001 — Gin Is the HTTP Runtime

**Status:** Accepted

Rationale:

- existing code investment;
- mature ecosystem;
- good performance;
- framework can add conventions without replacing Gin.

## ADR-002 — GORM Is the v1 ORM

**Status:** Accepted

Rationale:

- existing code investment;
- official drivers for required SQL targets;
- migrations/schema tooling can build around GORM while preserving escape hatches.

## ADR-003 — SQLite, PostgreSQL, and MySQL Are First-Class

**Status:** Accepted

Rationale:

- SQLite provides zero-infrastructure local/prototype workflows;
- PostgreSQL is a strong production default;
- MySQL supports a large existing deployment ecosystem.

Condition:

- official support requires conformance CI.

## ADR-004 — REST/JSON Is the Default; gRPC Is Optional

**Status:** Accepted

Rationale:

- React applications need a simple browser-friendly default;
- gRPC remains valuable for service boundaries;
- service-layer separation permits both.

## ADR-005 — React + TypeScript + Vite Is the Frontend

**Status:** Accepted

Rationale:

- existing React/TS application investment;
- TypeScript integrates naturally with generated OpenAPI contracts;
- Vite provides a modern dev/build foundation.

## ADR-006 — CLI and Code Generation Are Core Features

**Status:** Accepted

Rationale:

A Rails/Django/Laravel-like experience requires scaffolding and integrated workflows; a static starter alone does not satisfy the product goal.

## ADR-007 — Do Not Abstract GORM Behind a Universal ORM Interface

**Status:** Accepted

Rationale:

Preserve GORM's features and avoid recreating an ORM.

## ADR-008 — Cache Is Abstracted

**Status:** Accepted

Rationale:

Memory/Redis/disabled cache implementations have a narrow shared contract and materially improve local-development ergonomics.

## ADR-009 — Browser Secrets Are Forbidden

**Status:** Accepted

Rationale:

Any secret shipped to browser JavaScript is observable by the user and must be treated as public.

## ADR-010 — Runtime Plugins Are Compile-Time Go Modules

**Status:** Proposed

Rationale:

Preserve Go portability and type safety; avoid platform-specific runtime plugin behavior.

---

# 57. Open Decisions

These should be resolved before the public alpha:

1. framework/project name and CLI binary name;
2. organization/package naming;
3. exact migration DSL API;
4. MUI as default preset vs minimal default;
5. cookie authentication as default vs interactive choice;
6. exact OpenAPI-to-TypeScript generator;
7. whether frontend embedding is default in production;
8. whether repositories are generated for every resource or only when requested;
9. generic CRUD repository API;
10. migration checksum policy;
11. package manager default (`npm`, `pnpm`, or detected);
12. file naming conventions (`snake_case` vs Go-oriented naming for generated source);
13. default API prefix (`/api`, `/api/v1`, configurable);
14. public compatibility guarantees before v1;
15. license and governance model.

---

# 58. Success Metrics

The framework is succeeding when:

- a new developer can create and run a SQLite app with one installation and two commands;
- switching initial project choice to PostgreSQL/MySQL does not change application code;
- a generated CRUD resource spans database, API, OpenAPI, TypeScript, and React without manual type duplication;
- the generated project is understandable without knowing framework internals;
- production defaults include auth, logging, metrics, health, graceful shutdown, and security controls;
- advanced users can drop to Gin/GORM/raw SQL when needed;
- upgrades do not routinely destroy generated/user-edited code;
- framework applications compile and test consistently across supported database drivers.

---

# 59. Example “Golden Path”

```bash
# Install
go install github.com/<org>/<framework>/cmd/fw@latest

# Create
fw new inventory --database postgres --cache redis --auth cookie

cd inventory

# Start
fw dev

# Generate domain
fw make resource Product \
  sku:string:required,unique \
  name:string:required \
  price:decimal:required \
  active:bool:default=true

# Apply DB change
fw db migrate

# Inspect
fw routes

# Test
fw test

# Build
fw build
```

Expected result:

```text
dist/
└── inventory
```

or, for split deployment:

```text
dist/
├── inventory-api
└── frontend/
```

---

# 60. Conclusion

The project should be treated as a **framework extraction and productization effort**, not a template cleanup.

The existing backend already contains much of the production infrastructure. The existing CRUD monorepo already demonstrates the full-stack composition. The next leverage comes from:

1. extracting application-agnostic runtime packages;
2. defining firm conventions;
3. supporting SQLite/PostgreSQL/MySQL through a conformance-tested GORM database layer;
4. replacing manual frontend/backend contracts with generated OpenAPI → TypeScript artifacts;
5. building the CLI and generators;
6. integrating React/Vite development and production builds;
7. moving advanced batteries—jobs, events, scheduler, mail, storage, gRPC—behind stable modules after the core path is excellent.

The desired identity can be summarized as:

> **Rails/Laravel/Django-level cohesion and productivity, implemented with Go-level explicitness, Gin/GORM pragmatism, and a typed React frontend.**

---

# Appendix A — Proposed Public Runtime Sketch

```go
type App struct {
    // internal framework state
}

func New(opts ...Option) (*App, error)

func Run(application Application) error

func (a *App) Config() *config.Config
func (a *App) Router() *gin.Engine
func (a *App) DB() *gorm.DB
func (a *App) Cache() cache.Cache
func (a *App) Logger() *zap.Logger
func (a *App) Events() events.Bus
func (a *App) Queue() queue.Queue

func (a *App) OnStart(fn Hook)
func (a *App) OnStop(fn Hook)
```

Application:

```go
type Application interface {
    Register(*framework.App) error
}
```

---

# Appendix B — Proposed Module Sketch

```go
type Module interface {
    Register(*framework.App) error
}
```

Product module:

```go
type Module struct {
    service    *Service
    controller *Controller
}

func NewModule(app *framework.App) *Module {
    repo := repository.New[models.Product](app.DB())
    svc := NewService(repo)
    ctl := NewController(svc)

    return &Module{
        service: svc,
        controller: ctl,
    }
}

func (m *Module) Register(app *framework.App) error {
    r := app.Router().Group("/api/products")

    r.GET("", m.controller.Index)
    r.GET("/:id", m.controller.Show)
    r.POST("", m.controller.Create)
    r.PUT("/:id", m.controller.Update)
    r.DELETE("/:id", m.controller.Delete)

    return nil
}
```

---

# Appendix C — Configuration Validation Examples

Production should reject or loudly fail for cases such as:

```text
JWT secret too short
cookie auth without Secure under HTTPS production URL
wildcard CORS with credentials
trusted proxies = all without explicit opt-in
debug Gin mode in production
embedded browser API secret
SQLite path unwritable
pending required migrations
Redis selected but unreachable when required for auth/session semantics
```

---

# Appendix D — Reference Sources Reviewed

Project baselines:

- https://github.com/LAA-Software-Engineering/golang-rest-api-template
- https://github.com/LAA-Software-Engineering/crud-template-monorepo
- https://raw.githubusercontent.com/LAA-Software-Engineering/golang-rest-api-template/main/cmd/server/main.go
- https://raw.githubusercontent.com/LAA-Software-Engineering/golang-rest-api-template/main/pkg/api/router.go
- https://raw.githubusercontent.com/LAA-Software-Engineering/golang-rest-api-template/main/pkg/database/db.go
- https://raw.githubusercontent.com/LAA-Software-Engineering/golang-rest-api-template/main/pkg/cache/cache.go
- https://raw.githubusercontent.com/LAA-Software-Engineering/crud-template-monorepo/main/frontend/package.json
- https://raw.githubusercontent.com/LAA-Software-Engineering/crud-template-monorepo/main/frontend/src/services/api.ts
- https://raw.githubusercontent.com/LAA-Software-Engineering/crud-template-monorepo/main/frontend/src/types/index.ts

Technology references:

- https://gorm.io/docs/connecting_to_the_database.html
- https://vite.dev/guide/
- https://grpc.io/docs/languages/go/
