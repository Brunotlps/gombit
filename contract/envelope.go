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
type PageMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}
