// Package admin is Gombit's runtime generic admin (ADMIN-1 through ADMIN-3 /
// ADR-013).
//
// Feature packages register models explicitly:
//
//	admin.Register(app, Product{}, admin.Options{Slug: "products", ...})
//
// framework.New mounts the Huma introspection and data-plane routes, and
// the /admin/ SPA, only when cookie session auth is on (cfg.Auth.Mode ==
// cookie). JWT-only apps do not get admin routes. Admin requests enforce the
// registered permission keys; auth.User.IsSuperuser bypasses those checks.
//
// Options is the source of truth. After Register returns, handlers read
// stored values and GORM constructors — they do not reflect over arbitrary
// Go types. FieldsFrom and the empty-Fields default may use reflect only
// inside Register; both are registration-time conveniences.
package admin
