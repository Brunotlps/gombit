# Frontend MUI preset (`--ui mui`)

Default `gombit new` stays **minimal/headless** (C4). `--ui mui` is an
opt-in generated frontend preset — MUI never appears in the default
tree or in Gombit runtime Go packages.

```sh
gombit new demo --ui mui
gombit new demo --auth cookie --ui mui   # CSRF + MUI screens
```

Auth behavior (Bearer in-memory tokens, or cookie/CSRF) is independent
of the UI preset. See [frontend.md](frontend.md) and
[auth-cookie.md](auth-cookie.md). Walkthrough:
[examples/frontend-mui](../examples/frontend-mui/README.md).

## What gets scaffolded

- **Dependencies** (only when `--ui mui`): `@mui/material`,
  `@mui/icons-material`, `@emotion/react`, `@emotion/styled`. The
  scaffold does not add axios, date-fns, react-router-dom, or CRA.
- **`frontend/src/theme.ts`**: `createTheme` with primary `#1976d2`,
  secondary `#dc004e`, and `MuiButton.styleOverrides.root.textTransform
  = 'none'`. This file is omitted from the minimal tree.
- **`ThemeProvider` + `CssBaseline`** wrap the app in
  `frontend/src/app/providers.tsx`.
- **Layout**: MUI `AppBar` + `Toolbar` + `Typography` + `Button`. Logout
  keeps the Gombit session path (in-memory Bearer revoke, or cookie
  logout). No `localStorage` token storage and no ThemeToggle/dark mode.
- **Login**: `Paper` + `Typography` + `TextField` + `Button` + `Alert` +
  `CircularProgress` + React Hook Form. Same `/api/v1/auth/login` and
  `/register` handlers as the minimal preset.
- **List**: MUI `Table` / `TableContainer` with a loading
  `CircularProgress` and an empty state. Product scaffold is list+create
  only — no edit/delete actions that would need missing API routes.
- **Form**: dedicated `/products/new` (and `make resource` `/<slug>/new`)
  pages styled with `Paper` + `TextField` + `Button`. D10
  `applyContractErrors` / RHF `setError` still map `error.fields`.
- **Roboto** via a Google Fonts `<link>` in `frontend/index.html`.

`gombit make resource` reads `ui:` from `gombit.yaml` the same way it
reads `api_prefix:` and `database:`. When `ui: mui`, generated
`frontend/src/<feature>/list.tsx` and `form.tsx` use MUI Table/TextField
instead of raw `<table>` / `<input>`.

Access tokens stay **in memory**. `VITE_*` values are public; never put
JWT secrets in the frontend.
