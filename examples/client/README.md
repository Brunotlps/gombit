# TypeScript client example

Sample OpenAPI 3.1 document for the contract widget API, plus the output of
`gombit client generate`. Typed errors use the D10 envelope.

## Generate

From the repository root:

```sh
go run ./cmd/gombit client generate \
  --spec examples/client/openapi.json \
  --out examples/client/frontend/src/api/generated
```

## Typecheck

```sh
cd examples/client
npm install
npx tsc --noEmit
```

## Use

```ts
import {
  ContractError,
  createGombitClient,
  unwrap,
} from "./frontend/src/api/generated/client";

const client = createGombitClient({
  baseUrl: "http://127.0.0.1:8080",
  getAccessToken: () => undefined,
});

const widgets = await unwrap(await client.GET("/api/v1/widgets"));
```

See [`docs/client.md`](../../docs/client.md).
