package shared

import "net/http"

// Data is the D10 success envelope without meta: {"data": ...}. Same JSON
// shape as contract.Data (github.com/gombit-dev/gombit/contract) — defined
// independently here, not imported from there. contract.ErrorEnvelope is a
// huma.StatusError, so importing the contract package pulls in Huma
// regardless of whether the importer ever calls Huma directly; a Go
// implementation under benchmarks/apps/ that's sold as "no Huma" (the
// gin-gorm control) needs to match Gombit's wire format without linking
// Gombit's framework packages to do it, or the claim is false in the
// dependency graph even when it's true in the handler code.
type Data[T any] struct {
	Data T `json:"data"`
}

// DataMeta is the D10 success envelope with typed meta:
// {"data": ..., "meta": ...}.
type DataMeta[T any, M any] struct {
	Data T  `json:"data"`
	Meta *M `json:"meta,omitempty"`
}

// ErrorBody is the D10 error object nested under "error".
type ErrorBody struct {
	Code      string              `json:"code"`
	Message   string              `json:"message"`
	Fields    map[string][]string `json:"fields,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}

// ErrorEnvelope is the D10 error response body: {"error": {...}}.
type ErrorEnvelope struct {
	Status int       `json:"-"`
	Body   ErrorBody `json:"error"`
}

func (e *ErrorEnvelope) Error() string {
	if e == nil {
		return ""
	}
	return e.Body.Message
}

// NotFoundError, ConflictError, ValidationError, and InternalError mirror
// the D10 §41 category -> status mapping (contract.Category / StatusFor in
// github.com/gombit-dev/gombit/contract) without importing that package —
// same codes and statuses, independently implemented.
func NotFoundError(message string) *ErrorEnvelope {
	return &ErrorEnvelope{Status: http.StatusNotFound, Body: ErrorBody{Code: "not_found", Message: message}}
}

func ConflictError(message string) *ErrorEnvelope {
	return &ErrorEnvelope{Status: http.StatusConflict, Body: ErrorBody{Code: "conflict", Message: message}}
}

func ValidationError(message string, fields map[string][]string) *ErrorEnvelope {
	return &ErrorEnvelope{Status: http.StatusUnprocessableEntity, Body: ErrorBody{Code: "validation_error", Message: message, Fields: fields}}
}

func InternalError(message string) *ErrorEnvelope {
	return &ErrorEnvelope{Status: http.StatusInternalServerError, Body: ErrorBody{Code: "internal", Message: message}}
}
