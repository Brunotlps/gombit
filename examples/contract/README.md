# Contract example

Demonstrates Huma DTOs, D10 validation errors, success `meta`, §41
category errors (`not_found`), `/openapi.json`, and the `/docs` try-it-out UI
on `framework.App.API()`.

## Run

```sh
go run ./examples/contract
```

Open [http://127.0.0.1:8080/docs](http://127.0.0.1:8080/docs) for Swagger UI.
Try-it-out calls this same process. Raw Gin routes are not listed there.

```sh
curl -sS http://127.0.0.1:8080/openapi.json
go run ./cmd/gombit openapi generate \
  --url http://127.0.0.1:8080/openapi.json \
  --out /tmp/gombit-openapi.json
```

## List (data + page meta)

```sh
curl -sS http://127.0.0.1:8080/api/v1/widgets
```

## Get existing / missing

```sh
curl -sS http://127.0.0.1:8080/api/v1/widgets/widget-1
curl -sS http://127.0.0.1:8080/api/v1/widgets/missing
```

Missing id returns:

```json
{
  "error": {
    "code": "not_found",
    "message": "widget not found",
    "request_id": "..."
  }
}
```

## Valid create

```sh
curl -sS -X POST http://127.0.0.1:8080/api/v1/widgets \
  -H 'Content-Type: application/json' \
  -d '{"name":"Second widget","color":"green"}'
```

## Invalid body (D10 field errors)

```sh
curl -sS -X POST http://127.0.0.1:8080/api/v1/widgets \
  -H 'Content-Type: application/json' \
  -d '{}'
```

See [`docs/contract.md`](../../docs/contract.md) and
[`docs/openapi.md`](../../docs/openapi.md).
