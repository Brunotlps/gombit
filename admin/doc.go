// Package admin is Gombit's runtime generic admin (ADMIN-1 / ADR-013).
//
// Feature packages register models explicitly:
//
//	admin.Register(app, Product{}, admin.Options{Slug: "products", ...})
//
// framework.New mounts the Huma introspection and data-plane routes only when
// cookie session auth is on (cfg.Auth.Mode == cookie). JWT-only apps do not
// get admin routes. Until ADMIN-3, every admin request also requires
// auth.User.IsSuperuser.
//
// Options is the source of truth. After Register returns, handlers read
// stored values and GORM constructors — they do not reflect over arbitrary
// Go types. FieldsFrom and the empty-Fields default may use reflect only
// inside Register; both are registration-time conveniences.
package admin
