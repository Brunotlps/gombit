// Package auth is Gombit's runtime Bearer JWT surface (C3 / M5-2).
//
// Behavior lives here, not in generated apps. framework.New mounts the
// Huma routes when config.Auth.JWTSecret is set and a database is attached.
//
// Access tokens are short-lived JWTs. Refresh tokens are opaque, stored
// hashed, and rotated on each successful refresh. The generated SPA keeps
// both in memory — never localStorage or sessionStorage. Cookie/session
// mode is M5-3.
package auth
