# Frontend (MUI preset)

This directory is the split-deploy frontend root (build plan C5 / §3.2).
`gombit new --ui mui` writes a **Vite + React + TypeScript** app with the
opt-in MUI CRUD preset: `ThemeProvider` + `CssBaseline`, an AppBar layout,
Paper/TextField login, a product Table, and a dedicated create form.
Default `gombit new` stays **minimal/headless** (C4). See the framework
[`docs/frontend-mui.md`](https://github.com/gombit-dev/gombit/blob/main/docs/frontend-mui.md).
Bearer login/refresh is documented in the framework [`docs/auth.md`](https://github.com/gombit-dev/gombit/blob/main/docs/auth.md).
Cookie/CSRF mode (`--auth cookie`) is independent of the UI preset.

```sh
gombit dev
```

That starts the Go API, Vite HMR, and live `gombit client generate` into
`src/api/generated`. Vite proxies `/api`, `/openapi.json`, and `/docs`.

Optional single-binary production (split remains the default):

```sh
gombit build --embed
```

That runs the Vite production build, copies `dist/` into
`internal/web/static`, and compiles `./cmd/server`. See the framework
[`docs/build.md`](https://github.com/gombit-dev/gombit/blob/main/docs/build.md).

Public API origin:

```
VITE_API_URL
```

Empty (the `gombit new` / `gombit dev` default) means same-origin so the
Vite `/api` proxy works. For a split deploy, set the API **origin only**
(for example `http://127.0.0.1:8080`). OpenAPI path keys stay `/api/v1/...`;
`createAppClient` rewrites them to the live `GOMBIT_API_PREFIX` (injected
into `index.html` at serve time — not baked at `gombit new`). `VITE_*`
values are public. Do not put JWT secrets, database passwords, or other
server credentials here. Access tokens stay in memory — never
`localStorage` or `sessionStorage`.

The home page lists products via `unwrap(client.GET("/api/v1/products"))`.
`/products/new` is a React Hook Form create page; D10 field errors call
`setError` through `src/api/formErrors.ts`.

`src/api/generated` ships a placeholder product contract so `npm run
typecheck` / `npm run build` succeed immediately. `gombit client generate`
and `gombit dev` overwrite those files (they carry the generated banner).

`gombit make resource` writes React list/form pages under `src/<feature>/`
and refreshes `src/resources.tsx`. Those pages import types from
`src/api/generated` (no hand-written API DTOs). When `gombit.yaml` has
`ui: mui`, the pages use MUI Table/TextField instead of raw HTML.
Re-run client generate after adding routes.
