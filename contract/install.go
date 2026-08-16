package contract

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
)

// InstallOptions configures Huma error installation.
type InstallOptions struct {
	// RequestID reads the active request ID from a request context.
	RequestID func(context.Context) string
}

var installOnce sync.Once

// Install replaces Huma's default RFC 9457 Problem Details errors with the D10
// envelope. It is safe to call more than once; only the first call applies.
func Install(opts InstallOptions) {
	installOnce.Do(func() {
		requestID := opts.RequestID
		if requestID == nil {
			requestID = func(context.Context) string { return "" }
		}

		huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
			return newEnvelope(status, msg, "", errs...)
		}
		huma.NewErrorWithContext = func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
			id := ""
			if ctx != nil {
				id = requestID(ctx.Context())
			}
			return newEnvelope(status, msg, id, errs...)
		}
	})
}

func newEnvelope(status int, msg, requestID string, errs ...error) *ErrorEnvelope {
	fields := FieldsFromErrors(errs...)
	code, message := classifyError(status, msg, fields)
	return &ErrorEnvelope{
		status: status,
		Body: ErrorBody{
			Code:      code,
			Message:   message,
			Fields:    fields,
			RequestID: requestID,
		},
	}
}

func classifyError(status int, msg string, fields map[string][]string) (code, message string) {
	msg = strings.TrimSpace(msg)
	if isValidationFailure(status, msg, fields) {
		message = validationMessage
		if msg != "" && !strings.EqualFold(msg, "validation failed") {
			message = msg
		}
		return CodeValidationError, message
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	if msg == "" {
		msg = "request failed"
	}
	return statusCodeSlug(status), msg
}

func isValidationFailure(status int, msg string, fields map[string][]string) bool {
	// Any error carrying field details is treated as validation until M3-2 owns
	// application error categories (field-bearing domain errors may then use a
	// different code while keeping D10 fields).
	if len(fields) > 0 {
		return true
	}
	if status == http.StatusUnprocessableEntity {
		return true
	}
	if status == http.StatusBadRequest && strings.EqualFold(msg, "validation failed") {
		return true
	}
	return false
}

func statusCodeSlug(status int) string {
	text := strings.TrimSpace(http.StatusText(status))
	if text == "" {
		return "error"
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	slug := strings.Trim(b.String(), "_")
	if slug == "" {
		return "error"
	}
	return slug
}
