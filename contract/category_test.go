package contract

import (
	"net/http"
	"testing"
)

func TestCategoryStatusAndCodeTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cat        Category
		wantStatus int
		wantCode   string
	}{
		{CategoryValidation, http.StatusUnprocessableEntity, CodeValidationError},
		{CategoryAuthentication, http.StatusUnauthorized, "authentication"},
		{CategoryAuthorization, http.StatusForbidden, "authorization"},
		{CategoryNotFound, http.StatusNotFound, "not_found"},
		{CategoryConflict, http.StatusConflict, "conflict"},
		{CategoryRateLimited, http.StatusTooManyRequests, "rate_limited"},
		{CategoryDependencyUnavailable, http.StatusServiceUnavailable, "dependency_unavailable"},
		{CategoryInternal, http.StatusInternalServerError, "internal"},
		{Category("unknown"), http.StatusInternalServerError, "internal"},
	}
	for _, tt := range tests {
		t.Run(string(tt.cat), func(t *testing.T) {
			t.Parallel()
			if got := StatusFor(tt.cat); got != tt.wantStatus {
				t.Fatalf("StatusFor(%q) = %d, want %d", tt.cat, got, tt.wantStatus)
			}
			if got := CodeFor(tt.cat); got != tt.wantCode {
				t.Fatalf("CodeFor(%q) = %q, want %q", tt.cat, got, tt.wantCode)
			}
		})
	}
}

func TestNewConstructors(t *testing.T) {
	t.Parallel()

	err := NotFound("widget not found")
	if err.GetStatus() != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", err.GetStatus())
	}
	if err.Body.Code != "not_found" {
		t.Fatalf("code = %q, want not_found", err.Body.Code)
	}
	if err.Body.Message != "widget not found" {
		t.Fatalf("message = %q", err.Body.Message)
	}

	withFields := Conflict("name taken").WithFields(map[string][]string{
		"name": {"already exists"},
	})
	if withFields.Body.Code != "conflict" {
		t.Fatalf("field-bearing conflict code = %q, want conflict (not validation_error)", withFields.Body.Code)
	}
	if withFields.GetStatus() != http.StatusConflict {
		t.Fatalf("status = %d, want 409", withFields.GetStatus())
	}
	if len(withFields.Body.Fields["name"]) == 0 {
		t.Fatalf("fields missing: %#v", withFields.Body.Fields)
	}

	val := Validation("", map[string][]string{"email": {"invalid"}})
	if val.Body.Code != CodeValidationError {
		t.Fatalf("validation code = %q", val.Body.Code)
	}
	if val.Body.Message != validationMessage {
		t.Fatalf("validation message = %q", val.Body.Message)
	}

	tooLarge := PayloadTooLarge("")
	if tooLarge.GetStatus() != http.StatusRequestEntityTooLarge {
		t.Fatalf("payload too large status = %d, want 413", tooLarge.GetStatus())
	}
	if tooLarge.Body.Code != CodePayloadTooLarge {
		t.Fatalf("payload too large code = %q, want %s", tooLarge.Body.Code, CodePayloadTooLarge)
	}
	if tooLarge.Body.Message == "" {
		t.Fatal("payload too large message is empty")
	}
}
