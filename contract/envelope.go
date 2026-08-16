package contract

// Data is the D10 success envelope without meta: {"data": ...}.
type Data[T any] struct {
	Data T `json:"data"`
}

// DataMeta is the D10 success envelope with optional meta: {"data": ..., "meta": ...}.
type DataMeta[T any] struct {
	Data T   `json:"data"`
	Meta any `json:"meta,omitempty"`
}

// PageMeta is the conventional pagination meta object for collection responses.
type PageMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}
