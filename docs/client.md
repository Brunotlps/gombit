# TypeScript client generation

`gombit client generate` turns an OpenAPI 3.1 document into TypeScript types
and a thin `openapi-fetch` wrapper. Typed errors map to the D10 envelope.
This is D5: `openapi-typescript` + `openapi-fetch`, generation-only.

## Generate

```sh
go run ./cmd/gombit client generate \
  --spec openapi.json \
  --out frontend/src/api/generated
```

`--spec` defaults to `openapi.json`. `--out` defaults to
`frontend/src/api/generated` (design §23.3). The command prints created or
modified files and reminds you to add `openapi-fetch@0.17.0`. Schema types
come from `openapi-typescript@7.13.0`.

`--dry-run` prints the file list without writing and applies the same
overwrite rule as a real run: a user-owned file (no generated banner) is
refused unless `--force` is also set. `--force` overwrites a file that was
not produced by Gombit. Re-running replaces only files that carry the
generated banner.

Write the spec first with [`gombit openapi generate`](openapi.md) (or copy
`examples/client/openapi.json`).

## Output

| File | Role |
| --- | --- |
| `schema.ts` | Types from `openapi-typescript` |
| `error.ts` | D10 `error.{code,message,fields,request_id}` helpers |
| `client.ts` | `createGombitClient` + `unwrap` over `openapi-fetch` |

Do not hand-edit these files. Access tokens are supplied through
`getAccessToken` and stay in memory. Generated source never reads
`localStorage` or `sessionStorage`.

```ts
import {
  ContractError,
  createGombitClient,
  isD10ErrorBody,
  unwrap,
} from "./api/generated/client";

const client = createGombitClient({
  baseUrl: import.meta.env.VITE_API_URL ?? "http://127.0.0.1:8080",
  getAccessToken: () => accessToken,
});

const listed = await unwrap(await client.GET("/api/v1/widgets"));

try {
  await unwrap(await client.POST("/api/v1/widgets", { body: { name: "" } }));
} catch (err) {
  if (err instanceof ContractError) {
    // err.code, err.fields, err.requestId, err.status
  }
}

if (isD10ErrorBody(body)) {
  mapFieldsIntoForm(body.error.fields);
}
```

`VITE_API_URL` is public. Do not put secrets in `VITE_*` values.

## Example

`examples/client` ships a sample spec (the contract widget API) and the
generated client. See that README to typecheck it.

## What is not here yet

- CI drift check that fails when the spec or client is stale: M3-5
- Vite React skeleton that wires this client into forms: M5-1
