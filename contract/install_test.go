package contract

import (
	"net/http"
	"testing"
)

func TestClassifyErrorValidation(t *testing.T) {
	t.Parallel()

	code, message := classifyError(http.StatusUnprocessableEntity, "validation failed")
	if code != CodeValidationError {
		t.Fatalf("code = %q, want %q", code, CodeValidationError)
	}
	if message != validationMessage {
		t.Fatalf("message = %q, want %q", message, validationMessage)
	}
}

func TestStatusCodeSlug(t *testing.T) {
	t.Parallel()

	if got := statusCodeSlug(http.StatusNotFound); got != "not_found" {
		t.Fatalf("statusCodeSlug(404) = %q, want not_found", got)
	}
	if got := statusCodeSlug(0); got != "error" {
		t.Fatalf("statusCodeSlug(0) = %q, want error", got)
	}
}

func TestClassifyErrorUnexpected500IsInternal(t *testing.T) {
	t.Parallel()
	code, message := classifyError(http.StatusInternalServerError, "unexpected error occurred")
	if code != string(CategoryInternal) {
		t.Fatalf("code = %q, want %q", code, CategoryInternal)
	}
	if message != "unexpected error occurred" {
		t.Fatalf("message = %q, want unexpected error occurred (no driver string)", message)
	}
}

func TestClassifyErrorMapsHTTPStatusToCategoryCodes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status int
		want   string
	}{
		{http.StatusUnauthorized, string(CategoryAuthentication)},
		{http.StatusForbidden, string(CategoryAuthorization)},
		{http.StatusNotFound, string(CategoryNotFound)},
		{http.StatusConflict, string(CategoryConflict)},
		{http.StatusTooManyRequests, string(CategoryRateLimited)},
		{http.StatusServiceUnavailable, string(CategoryDependencyUnavailable)},
		{http.StatusInternalServerError, string(CategoryInternal)},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			code, _ := classifyError(tc.status, "")
			if code != tc.want {
				t.Fatalf("classifyError(%d) code = %q, want %q", tc.status, code, tc.want)
			}
		})
	}
}

func TestNewEnvelopeUnexpectedErrorOmitsFields(t *testing.T) {
	t.Parallel()
	env := newEnvelope(http.StatusInternalServerError, "unexpected error occurred", "req-1", errString("sql: connection refused"))
	if env.GetStatus() != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", env.GetStatus())
	}
	if env.Body.Code != string(CategoryInternal) {
		t.Fatalf("code = %q, want internal", env.Body.Code)
	}
	if len(env.Body.Fields) != 0 {
		t.Fatalf("fields = %#v, want empty (no driver string)", env.Body.Fields)
	}
}
