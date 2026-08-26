// Package shared holds the response-shape types common to every Go
// implementation under benchmarks/apps/ (currently gombit and gin-gorm) for
// the BENCH-1 canonical CRUD comparison (issue #141, benchmarks/docs/schema.md).
// Sharing these types between the two Go apps makes "same JSON shape" (one
// of the fairness checks issue #141 requires) true by construction instead
// of by two hand-maintained definitions staying in sync. Non-Go
// implementations (Django, Rails, Laravel, NestJS) can't share this package
// and must match benchmarks/docs/schema.md by hand instead.
package shared

import "time"

// PageMeta is the canonical /api/projects list-endpoint pagination meta
// shape: {"page","limit","total"}. Not contract.PageMeta (which uses
// per_page) — issue #141 explicitly specifies `limit` as both the query
// parameter and the meta field name for this benchmark's canonical API.
type PageMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// ProjectData is the canonical /api/projects response shape
// (benchmarks/docs/schema.md "Canonical CRUD API"). OwnerName is the
// preloaded owner relationship flattened into the response — its presence
// is what the N+1 fairness check verifies: a handler that didn't preload
// the owner would have to make a second query per row to populate it.
type ProjectData struct {
	ID          uint      `json:"id"`
	OwnerID     uint      `json:"owner_id"`
	OwnerName   string    `json:"owner_name"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
