# Contract example

Demonstrates Huma DTO conventions and D10 validation errors on
`framework.App.API()`.

## Run

```sh
go run ./examples/contract
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

Example response:

```json
{
  "error": {
    "code": "validation_error",
    "message": "The request contains invalid fields.",
    "fields": {
      "name": ["expected required property name to be present"]
    },
    "request_id": "..."
  }
}
```

See [`docs/contract.md`](../../docs/contract.md).
