package scenario

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// RunBenchmark runs the five framework-tax benchmark scenarios (plaintext,
// JSON, path parameter, valid POST, invalid POST) against stack.Handler as
// b.Run sub-benchmarks, reporting ns/op, B/op, and allocs/op for each — of
// the handler alone, not of the benchmark harness. It is shared by every row
// (benchmarks/micro/{nethttp,gin,huma,gombit}) so the sub-benchmark set and
// the request-timing methodology can't drift between packages.
//
// Every scenario builds its *http.Request once, outside the timed loop, and
// reuses it: none of the four stacks' middleware mutates a request in a way
// that corrupts reuse (verified — Gombit's XSS-sanitization middleware, the
// only header/body-mutating middleware in the runtime stack, replaces
// Body/Content-Length on the WithContext-derived request Gin's own machinery
// hands to handlers, not the shared object this loop holds; every
// request-derived context change goes through http.Request.WithContext,
// which returns a new Request rather than mutating the original; confirmed
// empirically with a 30-iteration POST reuse loop against a full
// framework.App before relying on it here). Skipping this hoist and
// building a fresh request every iteration measured the harness, not the
// handler: on a trivial "write a canned string" handler, per-iteration
// httptest.NewRequest+NewRecorder construction cost several times what the
// handler itself allocated, which — being roughly constant across all four
// stacks — compressed the very ratios this benchmark exists to report.
//
// GET scenarios have no body to manage. POST scenarios can't reuse the
// *http.Response body reader itself (an io.Reader that's been drained can't
// be un-drained), so each iteration replaces Body with a fresh
// io.NopCloser over the same payload bytes and resets ContentLength — cheap
// relative to constructing a whole new *http.Request via httptest.NewRequest
// (which re-parses the URL, rebuilds the request line, etc.).
func RunBenchmark(b *testing.B, stack Stack) {
	b.Helper()

	b.Run("plaintext", func(b *testing.B) {
		runGET(b, stack.Handler, "/plaintext")
	})
	b.Run("json", func(b *testing.B) {
		runGET(b, stack.Handler, "/json")
	})
	b.Run("path-param", func(b *testing.B) {
		runGET(b, stack.Handler, "/users/user-42")
	})
	b.Run("valid-post", func(b *testing.B) {
		runPOST(b, stack.Handler, ValidCreateUserBody, statusOKOrCreated)
	})
	b.Run("invalid-post", func(b *testing.B) {
		runPOST(b, stack.Handler, InvalidCreateUserBody, statusUnprocessableEntity)
	})
}

func runGET(b *testing.B, handler http.Handler, path string) {
	b.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("GET %s status = %d, want %d; body: %s", path, response.Code, http.StatusOK, response.Body.String())
		}
	}
}

func runPOST(b *testing.B, handler http.Handler, body string, wantStatus func(int) bool) {
	b.Helper()
	request := httptest.NewRequest(http.MethodPost, "/users", nil)
	request.Header.Set("Content-Type", "application/json")
	payload := []byte(body)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		request.Body = io.NopCloser(bytes.NewReader(payload))
		request.ContentLength = int64(len(payload))

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if !wantStatus(response.Code) {
			b.Fatalf("POST /users status = %d; body: %s", response.Code, response.Body.String())
		}
	}
}

func statusOKOrCreated(code int) bool {
	return code == http.StatusOK || code == http.StatusCreated
}

func statusUnprocessableEntity(code int) bool {
	return code == http.StatusUnprocessableEntity
}
