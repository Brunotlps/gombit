package scenario

import (
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
// GET scenarios build their *http.Request once, outside the timed loop, and
// reuse it: none of the four stacks' middleware mutates a body-less incoming
// request in place (verified — Gombit's XSS-sanitization middleware, the
// only header-mutating middleware in the runtime stack, short-circuits
// whenever the request body is nil/http.NoBody; every other request-derived
// state change goes through http.Request.WithContext, which returns a new
// Request rather than mutating the shared one). Skipping this hoist and
// building a fresh request every iteration measured the harness, not the
// handler: on a trivial "write a canned string" handler, per-iteration
// httptest.NewRequest+NewRecorder construction cost several times what the
// handler itself allocated, which — being roughly constant across all four
// stacks — compressed the very ratios this benchmark exists to report.
//
// POST scenarios can't do this: their request bodies are io.Readers that
// get drained by the handler, so reusing one request across iterations
// would silently serve an empty body from the second iteration on. Those
// still build a fresh request per iteration via Do.
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
		runPOST(b, stack.Handler, ValidCreateUserBody)
	})
	b.Run("invalid-post", func(b *testing.B) {
		runPOST(b, stack.Handler, InvalidCreateUserBody)
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
		if response.Code >= 500 {
			b.Fatalf("GET %s status = %d, want < 500; body: %s", path, response.Code, response.Body.String())
		}
	}
}

func runPOST(b *testing.B, handler http.Handler, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := Do(handler, http.MethodPost, "/users", body)
		if response.Code >= 500 {
			b.Fatalf("POST /users status = %d, want < 500; body: %s", response.Code, response.Body.String())
		}
	}
}
