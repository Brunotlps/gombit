package contract

import (
	"net/http"
	"testing"
)

func TestClassifyErrorValidation(t *testing.T) {
	t.Parallel()

	code, message := classifyError(http.StatusUnprocessableEntity, "validation failed", map[string][]string{
		"name": {"required"},
	})
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
