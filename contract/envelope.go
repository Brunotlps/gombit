package contract

// Data is the D10 success envelope without meta: {"data": ...}.
type Data[T any] struct {
	Data T `json:"data"`
}

// DataMeta is the D10 success envelope with typed meta:
// {"data": ..., "meta": ...}.
//
// Use a concrete meta type so OpenAPI emits a real schema (not empty object).
// Prefer a pointer meta value so omitempty drops absent meta:
//
//	contract.DataMeta[[]Widget, PageMeta]{Data: items, Meta: &contract.PageMeta{...}}
//
// A non-nil zero PageMeta still serializes (omitempty does not treat structs as empty).
type DataMeta[T any, M any] struct {
	Data T  `json:"data"`
	Meta *M `json:"meta,omitempty"`
}

// PageMeta is the conventional pagination meta object for collection responses.
// It is the v0.1 typed default for list endpoints; other meta shapes can use the
// second type parameter on DataMeta when needed.
//
// Generated list handlers and the admin data plane honor page/per_page with
// ClampPage defaults: page 1, per_page 20, max 100. Meta.Total is the matching
// row count, not len(data).
type PageMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

const (
	// DefaultPage is the 1-based page used when page is missing or < 1.
	DefaultPage = 1
	// DefaultPerPage is the page size used when per_page is missing or < 1.
	DefaultPerPage = 20
	// MaxPerPage is the upper clamp for per_page (admin and generated lists).
	MaxPerPage = 100
)

// ClampPage returns a 1-based page and a per-page size using the v0.1 defaults.
// Values less than 1 become DefaultPage / DefaultPerPage; per_page above
// MaxPerPage is clamped to MaxPerPage.
func ClampPage(page, perPage int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

// PageOffset returns the 0-based OFFSET for a 1-based page and per-page size.
// Callers should pass values already returned by ClampPage.
func PageOffset(page, perPage int) int {
	return (page - 1) * perPage
}
