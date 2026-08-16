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

var (
	installOnce    sync.Once
	requestIDFrom  func(context.Context) string
	requestIDMu    sync.RWMutex
)

// Install replaces Huma's default RFC 9457 Problem Details errors with the D10
// envelope. It is safe to call more than once; only the first call applies.
//
// Install only classifies errors built through Huma's NewError hooks (request
// validation). Application errors from New / NotFound / etc. set status and
// code via StatusFor / CodeFor and are not reclassified.
func Install(opts InstallOptions) {
	installOnce.Do(func() {
		requestID := opts.RequestID
		if requestID == nil {
			requestID = func(context.Context) string { return "" }
		}
		requestIDMu.Lock()
		requestIDFrom = requestID
		requestIDMu.Unlock()

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

// WithContext returns err with request_id filled from Install's RequestID
// reader when available. Application handlers should wrap category errors:
//
//	return nil, contract.WithContext(ctx, contract.NotFound("missing"))
func WithContext(ctx context.Context, err *ErrorEnvelope) *ErrorEnvelope {
	if err == nil {
		return nil
	}
	requestIDMu.RLock()
	fn := requestIDFrom
	requestIDMu.RUnlock()
	if fn == nil || ctx == nil {
		return err
	}
	if id := fn(ctx); id != "" {
		return err.WithRequestID(id)
	}
	return err
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
	// Huma Install path only: field details or 422/validation-failed indicate
	// request validation. Application constructors (New/NotFound/...) bypass
	// this function entirely.
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
