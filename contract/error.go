package contract

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// CodeValidationError is the stable D10 code for request validation failures.
const CodeValidationError = "validation_error"

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

var _ huma.StatusError = (*ErrorEnvelope)(nil)
