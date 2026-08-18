// Package build implements `gombit build --embed`: Vite production build,
// collectstatic into internal/web/static, and `go build` of a single binary
// that serves API + static + SPA fallback. Split deploy stays the default
// (C5); embedding is opt-in.
package build
