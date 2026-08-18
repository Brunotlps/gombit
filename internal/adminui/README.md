# Framework-owned admin SPA (ADMIN-2)

Vite + React + TypeScript app served from `go:embed` under `/admin/`.
Feature packages still only call `admin.Register` — there are **no**
per-model React files.

## Rebuild the committed `dist/`

`dist/` is committed so `go test` and `go run ./examples/admin` work
without npm at compile time (`go:embed` cannot use `..` and cannot embed
an empty directory).

```sh
cd internal/adminui
npm ci
npm run typecheck
npm run build
```

Commit the resulting `dist/` with the source change. Do not copy this
tree into generated `frontend/`. Do not add `--admin` to
`gombit make resource`.

`base` is `/admin/` so hashed assets are `/admin/assets/…`. Auth is
cookie + CSRF only. Tokens and CSRF stay out of `localStorage` /
`sessionStorage`. `VITE_API_URL` is empty (same-origin `/api/v1`).
