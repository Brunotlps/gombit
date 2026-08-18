# Frontend MUI preset example

MUI is a **generated** frontend preset, not a Gombit runtime package.
This directory does not commit a Vite tree. Scaffold one locally:

```sh
gombit new demo --ui mui
cd demo
gombit dev
```

`--ui mui` is opt-in. Default `gombit new demo` stays minimal/headless
and must not contain `@mui/` in `frontend/package.json`.

Auth is independent of UI. Combine presets when you want both CSRF and
MUI screens:

```sh
gombit new demo --auth cookie --ui mui
```

See [`docs/frontend-mui.md`](../../docs/frontend-mui.md).

## Screens (`gombit new demo --ui mui`)

| Route | UI |
| --- | --- |
| `/login` | Centered MUI `Paper` with `TextField`s, `Alert`, `CircularProgress` on submit, Log in + Create account |
| `/` (product list) | `AppBar` layout; MUI `Table` of id/name/price with loading spinner and empty state |
| `/products/new` | Dedicated create page (`Paper` + `TextField` + Create). Not a modal. |

`frontend/src/theme.ts` sets primary `#1976d2`, secondary `#dc004e`, and
turns off button `textTransform`. `AppProviders` wraps the tree in
`ThemeProvider` + `CssBaseline`.

`gombit make resource Book title:string:required` in a `ui: mui` app
writes MUI Table/TextField pages under `frontend/src/book/` that still
import generated OpenAPI types and call `applyContractErrors`.
