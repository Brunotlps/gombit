# Admin example (ADMIN-1 + ADMIN-2)

Minimal `framework.App` with `Config.Auth.Mode = config.AuthModeCookie`,
SQLite, and an explicit `admin.Register` of a small `widget.Widget` model.
`framework.New` mounts empty admin Huma routes and the framework-owned
SPA under `/admin/` in cookie mode; this example registers one model from
the feature package. JWT-only apps do not get these routes.

See [`docs/admin.md`](../../docs/admin.md) and
[ADR-013](../../docs/adr/013-runtime-generic-admin.md).

## Run

```sh
go run ./examples/admin
```

Admin UI: [http://127.0.0.1:8082/admin/](http://127.0.0.1:8082/admin/).
Sign in with the seeded superuser
`admin@example.com` / `correct-horse-battery-staple`. The widgets screen
appears because `widget.RegisterAdmin` called `admin.Register` — there is
no per-model React file.

Interactive docs: [http://127.0.0.1:8082/docs](http://127.0.0.1:8082/docs).

`/auth/register` never sets `IsSuperuser`. When the example is pointed at
a file-backed SQLite DSN, use `gombit createsuperuser` (see below).

## E2E (meta + one CRUD cycle)

A cookie jar is required. CSRF must be double-submitted on POST/PATCH/DELETE.

```sh
JAR=$(mktemp)

CSRF_BODY=$(curl -sS -c "$JAR" -b "$JAR" http://127.0.0.1:8082/api/v1/auth/csrf)
CSRF=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["csrf_token"])' <<<"$CSRF_BODY")

curl -sS -c "$JAR" -b "$JAR" -X POST http://127.0.0.1:8082/api/v1/auth/login \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"email":"admin@example.com","password":"correct-horse-battery-staple"}'

# catalog (data.models is required; empty array when nothing is registered)
curl -sS -c "$JAR" -b "$JAR" http://127.0.0.1:8082/api/v1/admin/meta
curl -sS -c "$JAR" -b "$JAR" http://127.0.0.1:8082/api/v1/admin/meta/widgets

# create / list / detail / update / delete
CREATED=$(curl -sS -c "$JAR" -b "$JAR" -X POST http://127.0.0.1:8082/api/v1/admin/resources/widgets \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Wrench","sku":"w-1"}')
ID=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["id"])' <<<"$CREATED")

curl -sS -c "$JAR" -b "$JAR" "http://127.0.0.1:8082/api/v1/admin/resources/widgets?search=Wrench&ordering=name"
curl -sS -c "$JAR" -b "$JAR" "http://127.0.0.1:8082/api/v1/admin/resources/widgets/$ID"
curl -sS -c "$JAR" -b "$JAR" -X PATCH "http://127.0.0.1:8082/api/v1/admin/resources/widgets/$ID" \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"sku":"w-1a"}'
curl -sS -c "$JAR" -b "$JAR" -X DELETE "http://127.0.0.1:8082/api/v1/admin/resources/widgets/$ID" \
  -H "X-CSRF-Token: $CSRF"

rm -f "$JAR"
```

Anonymous calls return D10 `authentication` (401). A normal registered user
(not a superuser) gets D10 `authorization` (403). Unknown slugs/ids are
`not_found`. A disabled action is `authorization` for an authenticated
superuser.

## Create a superuser against a file DSN

When the example is pointed at a file-backed SQLite DSN instead of the
in-memory default, use `gombit createsuperuser`:

```sh
GOMBIT_DATABASE_DRIVER=sqlite \
GOMBIT_DATABASE_DSN='file:admin-example.db?cache=shared&_fk=1' \
GOMBIT_JWT_SECRET='dev-only-example-jwt-secret-not-for-prod' \
GOMBIT_AUTH_MODE=cookie \
go run ./cmd/gombit createsuperuser --no-input \
  --email admin@example.com --password correct-horse-battery-staple
```
