package contract

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// CodeValidationError is the stable D10 code for request validation failures.
const CodeValidationError = "validation_error"

// CodePayloadTooLarge is the D10 code for a body that exceeded a framework
// buffer cap (HTTP 413). This is not a §41 category.
const CodePayloadTooLarge = "payload_too_large"

const validationMessage = "The request contains invalid fields."

// ErrorBody is the D10 error object nested under "error".
type ErrorBody struct {
	Code      string              `json:"code"`
	Message   string              `json:"message"`
	Fields    map[string][]string `json:"fields,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}

// ErrorEnvelope is the D10 error response body and a huma.StatusError.
//
// It intentionally does not implement huma.ContentTypeFilter so responses keep
// Content-Type application/json instead of application/problem+json.
type ErrorEnvelope struct {
	status int
	Body   ErrorBody `json:"error"`
}

// Error satisfies the error interface.
func (e *ErrorEnvelope) Error() string {
	if e == nil {
		return ""
	}
	return e.Body.Message
}

// GetStatus returns the HTTP status for this error.
func (e *ErrorEnvelope) GetStatus() int {
	if e == nil || e.status == 0 {
		return http.StatusInternalServerError
	}
	return e.status
}

// WithRequestID returns a shallow copy with request_id set (for tests/helpers).
func (e *ErrorEnvelope) WithRequestID(requestID string) *ErrorEnvelope {
	if e == nil {
		return nil
	}
	out := *e
	out.Body.RequestID = requestID
	return &out
}

// WithFields returns a shallow copy with D10 fields set. The error code stays
// the category code (not forced to validation_error).
func (e *ErrorEnvelope) WithFields(fields map[string][]string) *ErrorEnvelope {
	if e == nil {
		return nil
	}
	out := *e
	out.Body.Fields = fields
	return &out
}

// New builds a D10 ErrorEnvelope from a §41 category and message.
//
// Callers should wrap with WithContext so request_id is populated:
//
//	return nil, contract.WithContext(ctx, contract.New(contract.CategoryNotFound, "missing"))
func New(cat Category, message string) *ErrorEnvelope {
	return &ErrorEnvelope{
		status: StatusFor(cat),
		Body: ErrorBody{
			Code:    CodeFor(cat),
			Message: message,
		},
	}
}

// Validation returns a validation category error (HTTP 422, code validation_error).
// Wrap with WithContext in handlers to attach request_id.
func Validation(message string, fields map[string][]string) *ErrorEnvelope {
	if message == "" {
		message = validationMessage
	}
	return New(CategoryValidation, message).WithFields(fields)
}

// Authentication returns an authentication category error (HTTP 401).
// Wrap with WithContext in handlers to attach request_id.
func Authentication(message string) *ErrorEnvelope {
	return New(CategoryAuthentication, message)
}

// Authorization returns an authorization category error (HTTP 403).
// Wrap with WithContext in handlers to attach request_id.
func Authorization(message string) *ErrorEnvelope {
	return New(CategoryAuthorization, message)
}

// NotFound returns a not_found category error (HTTP 404).
// Wrap with WithContext in handlers to attach request_id.
func NotFound(message string) *ErrorEnvelope {
	return New(CategoryNotFound, message)
}

// Conflict returns a conflict category error (HTTP 409).
// Wrap with WithContext in handlers to attach request_id.
func Conflict(message string) *ErrorEnvelope {
	return New(CategoryConflict, message)
}

// RateLimited returns a rate_limited category error (HTTP 429).
// Wrap with WithContext in handlers to attach request_id.
func RateLimited(message string) *ErrorEnvelope {
	return New(CategoryRateLimited, message)
}

// PayloadTooLarge returns a D10 error for a request body that exceeded a
// framework buffer cap (HTTP 413). This is not a §41 category; the sanitizer
// bound is not the deferred body-size middleware.
func PayloadTooLarge(message string) *ErrorEnvelope {
	if message == "" {
		message = "The request body is too large."
	}
	return &ErrorEnvelope{
		status: http.StatusRequestEntityTooLarge,
		Body: ErrorBody{
			Code:    CodePayloadTooLarge,
			Message: message,
		},
	}
}

// DependencyUnavailable returns a dependency_unavailable error (HTTP 503).
// Wrap with WithContext in handlers to attach request_id.
func DependencyUnavailable(message string) *ErrorEnvelope {
	return New(CategoryDependencyUnavailable, message)
}

// Internal returns an internal category error (HTTP 500).
// Wrap with WithContext in handlers to attach request_id.
func Internal(message string) *ErrorEnvelope {
	return New(CategoryInternal, message)
}

var _ huma.StatusError = (*ErrorEnvelope)(nil)
