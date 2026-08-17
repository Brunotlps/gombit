# Frontend (minimal React skeleton)

`gombit new` writes a Vite + React + TypeScript app under `frontend/`.
The default UI is **minimal/headless** (C4). `--ui mui` is recorded in
`gombit.yaml` only; the MUI CRUD preset is [M5-4]. Bearer login, refresh,
and protected routes are [M5-2].

See also [cli.md](cli.md) (`gombit new`, `gombit dev`) and
[client.md](client.md) (TypeScript client generation).

## Stack

- Vite + `@vitejs/plugin-react`
- React + TypeScript
- React Router (`BrowserRouter`)
- generated `openapi-typescript` + `openapi-fetch` client
- React Hook Form with D10 `error.fields` mapping

Package manager D6: the scaffolded `package.json` works with npm (CI uses
Node 22). `gombit dev` prefers pnpm when it is available.

## Layout

```text
frontend/src/
├── main.tsx
├── app/
│   ├── providers.tsx   # API client context
│   └── router.tsx      # /, /products/new, generated resource routes
├── api/
│   ├── client.ts       # createAppClient + useApiClient
│   ├── formErrors.ts   # D10 fields → RHF setError
│   └── generated/      # schema.ts, client.ts, error.ts
├── auth/
│   └── session.ts      # in-memory access token
├── layouts/
│   └── AppLayout.tsx
├── pages/
│   ├── ProductListPage.tsx
│   └── ProductFormPage.tsx
└── resources.tsx       # rewritten by gombit make resource
```

## Talking to the API

The home page calls `unwrap(client.GET("/api/v1/products"))`. Create uses
`client.POST("/api/v1/products", { body })`. Paths come from the generated
OpenAPI types — do not hand-write DTOs.

`createGombitClient` `baseUrl` is `import.meta.env.VITE_API_URL` (public).
Empty means same-origin so the Vite `/api` proxy used by `gombit dev`
works. For a split deploy, set the API **origin only** (for example
`http://127.0.0.1:8080`); OpenAPI paths already include `/api/v1`.

`VITE_*` values are baked into the browser bundle. Never put JWT secrets,
database passwords, or other server credentials there.

## Access token (in memory)

`src/auth/session.ts` holds the access token in a module variable.
`getAccessToken` is passed into `createGombitClient` and may return
`undefined`. Login, refresh rotation, and protected routes are M5-2.
Generated source never reads `localStorage` or `sessionStorage`.

## D10 field errors

`src/api/formErrors.ts` maps `ContractError.fields` or a D10 error body
onto React Hook Form:

```ts
try {
  await unwrap(await client.POST("/api/v1/products", { body }));
} catch (err: unknown) {
  if (!applyContractErrors(setError, err)) {
    // non-field error (message / request_id)
  }
}
```

A body like `{"error":{"code":"validation_error","message":"...","fields":{"name":["required"]}}}`
calls `setError("name", { type: "server", message: "required" })`. Do not
invent another error shape.

## Generated client placeholder

`frontend/src/api/generated` ships a placeholder product contract so
`npm run typecheck` / `npm run build` succeed right after `gombit new`.
Those files carry the generated banner. `gombit client generate` and
`gombit dev` overwrite them from the live OpenAPI 3.1 document.

After `gombit make resource`, regenerate the client so new paths exist:

```sh
gombit client generate --spec openapi.json
# or
gombit dev
```

## What is not here yet

- Bearer login, refresh, protected routes: M5-2
- Cookie/session + CSRF (`--auth cookie`): M5-3
- MUI CRUD preset (`--ui mui`): M5-4
- `go:embed` single-binary SPA: M5-5
