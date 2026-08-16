package contract

import "net/http"

// Category is a draft §41 framework error category. Transport adapters map
// categories to HTTP status codes via StatusFor; they do not leak into service
// layer return types as raw status integers.
type Category string

const (
	CategoryValidation             Category = "validation"
	CategoryAuthentication         Category = "authentication"
	CategoryAuthorization          Category = "authorization"
	CategoryNotFound               Category = "not_found"
	CategoryConflict               Category = "conflict"
	CategoryRateLimited            Category = "rate_limited"
	CategoryDependencyUnavailable  Category = "dependency_unavailable"
	CategoryInternal               Category = "internal"
)

var categoryStatus = map[Category]int{
	CategoryValidation:            http.StatusUnprocessableEntity,
	CategoryAuthentication:        http.StatusUnauthorized,
	CategoryAuthorization:         http.StatusForbidden,
	CategoryNotFound:              http.StatusNotFound,
	CategoryConflict:              http.StatusConflict,
	CategoryRateLimited:           http.StatusTooManyRequests,
	CategoryDependencyUnavailable: http.StatusServiceUnavailable,
	CategoryInternal:              http.StatusInternalServerError,
}

// StatusFor returns the HTTP status for a §41 category. Unknown categories map
// to 500 Internal Server Error.
func StatusFor(cat Category) int {
	if status, ok := categoryStatus[cat]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// CodeFor returns the stable D10 error.code for a category. Validation keeps
// the M3-1 / D10 name "validation_error"; other categories use the category
// string. Unknown categories map to "internal".
func CodeFor(cat Category) string {
	if cat == CategoryValidation {
		return CodeValidationError
	}
	if _, ok := categoryStatus[cat]; ok {
		return string(cat)
	}
	return string(CategoryInternal)
}
