package scenario

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Stack is one row of the framework-tax matrix under test.
type Stack struct {
	Name    string
	Handler http.Handler
	// Envelope is true for stacks that wrap JSON responses in
	// SuccessEnvelope (Huma, Gombit); false for stacks that return the
	// resource bare (net/http, Gin) — each idiomatic to its own stack.
	Envelope bool
}

// Assert exercises the five framework-tax benchmark scenarios against
// stack.Handler and fails tb if the stack doesn't implement them correctly.
// It decodes and checks JSON structurally (not by substring), so a handler
// that stops doing real JSON serialization work can't silently keep passing:
// a benchmark over a broken or short-circuited handler is meaningless.
func Assert(tb testing.TB, stack Stack) {
	tb.Helper()

	assertPlaintext(tb, stack)
	assertJSONMessage(tb, stack)
	assertGetUser(tb, stack)
	assertValidPost(tb, stack)
	assertInvalidPost(tb, stack)
}

func assertPlaintext(tb testing.TB, stack Stack) {
	tb.Helper()

	response := Do(stack.Handler, http.MethodGet, "/plaintext", "")
	if response.Code != http.StatusOK {
		tb.Fatalf("%s: GET /plaintext status = %d, want %d", stack.Name, response.Code, http.StatusOK)
	}
	if got := response.Body.String(); got != "Hello, World!" {
		tb.Fatalf("%s: GET /plaintext body = %q, want %q", stack.Name, got, "Hello, World!")
	}
}

func assertJSONMessage(tb testing.TB, stack Stack) {
	tb.Helper()

	response := Do(stack.Handler, http.MethodGet, "/json", "")
	if response.Code != http.StatusOK {
		tb.Fatalf("%s: GET /json status = %d, want %d; body: %s", stack.Name, response.Code, http.StatusOK, response.Body.String())
	}

	message := decodeMessage(tb, stack, response.Body.Bytes())
	if message != "Hello, World!" {
		tb.Fatalf("%s: GET /json decoded message = %q, want %q", stack.Name, message, "Hello, World!")
	}
}

func assertGetUser(tb testing.TB, stack Stack) {
	tb.Helper()

	response := Do(stack.Handler, http.MethodGet, "/users/user-42", "")
	if response.Code != http.StatusOK {
		tb.Fatalf("%s: GET /users/user-42 status = %d, want %d; body: %s", stack.Name, response.Code, http.StatusOK, response.Body.String())
	}

	user := decodeUser(tb, stack, response.Body.Bytes())
	if user.ID != "user-42" {
		tb.Fatalf("%s: GET /users/user-42 decoded id = %q, want %q", stack.Name, user.ID, "user-42")
	}
	if user.Name == "" {
		tb.Fatalf("%s: GET /users/user-42 decoded name is empty", stack.Name)
	}
}

func assertValidPost(tb testing.TB, stack Stack) {
	tb.Helper()

	response := Do(stack.Handler, http.MethodPost, "/users", ValidCreateUserBody)
	if response.Code != http.StatusOK && response.Code != http.StatusCreated {
		tb.Fatalf("%s: POST /users (valid) status = %d, want 200 or 201; body: %s", stack.Name, response.Code, response.Body.String())
	}

	user := decodeUser(tb, stack, response.Body.Bytes())
	if user.Name != "Ada Lovelace" {
		tb.Fatalf("%s: POST /users (valid) decoded name = %q, want %q", stack.Name, user.Name, "Ada Lovelace")
	}
	if user.Email != "ada@example.com" {
		tb.Fatalf("%s: POST /users (valid) decoded email = %q, want %q", stack.Name, user.Email, "ada@example.com")
	}
	if user.ID == "" {
		tb.Fatalf("%s: POST /users (valid) decoded id is empty", stack.Name)
	}
}

// assertInvalidPost checks that every stack rejects the payload with exactly
// 422 Unprocessable Entity — what all four stacks actually return for this
// payload (well-formed JSON, invalid field values): net/http and Gin return
// it explicitly from their own validation; Huma and Gombit normalize
// validation failures to 422 via contract.isValidationFailure regardless of
// error-body shape. A looser `4xx` range would also accept 404/405, which a
// broken route registration (wrong path, wrong method) would produce just as
// easily as a working validator — silently turning a routing bug into a
// passing test. It deliberately does not assert a specific error envelope
// shape: contract.Install (M3-2) replaces Huma's package-level huma.NewError
// process-wide the first time any framework.App boots, so a bare Huma+Gin
// stack sharing a process with a Gombit stack would have an error shape that
// depends on run order. Each stack here runs in its own package/process
// (that's the point of benchmarks/micro/{...} being separate directories),
// which sidesteps that for status codes, but this assertion still doesn't
// assert an exact shape since it's shared across stacks that intentionally
// use different error mappings (see docs/spikes/M0-2_HUMA_GIN_SPIKE.md).
func assertInvalidPost(tb testing.TB, stack Stack) {
	tb.Helper()

	response := Do(stack.Handler, http.MethodPost, "/users", InvalidCreateUserBody)
	if response.Code != http.StatusUnprocessableEntity {
		tb.Fatalf("%s: POST /users (invalid) status = %d, want %d", stack.Name, response.Code, http.StatusUnprocessableEntity)
	}
	if !json.Valid(response.Body.Bytes()) {
		tb.Fatalf("%s: POST /users (invalid) body is not valid JSON: %s", stack.Name, response.Body.String())
	}
}

func decodeUser(tb testing.TB, stack Stack, body []byte) BenchUser {
	tb.Helper()

	if stack.Envelope {
		var envelope SuccessEnvelope[BenchUser]
		if err := json.Unmarshal(body, &envelope); err != nil {
			tb.Fatalf("%s: decode enveloped user response: %v; body: %s", stack.Name, err, body)
		}
		return envelope.Data
	}

	var user BenchUser
	if err := json.Unmarshal(body, &user); err != nil {
		tb.Fatalf("%s: decode bare user response: %v; body: %s", stack.Name, err, body)
	}
	return user
}

func decodeMessage(tb testing.TB, stack Stack, body []byte) string {
	tb.Helper()

	if stack.Envelope {
		var envelope SuccessEnvelope[map[string]string]
		if err := json.Unmarshal(body, &envelope); err != nil {
			tb.Fatalf("%s: decode enveloped json response: %v; body: %s", stack.Name, err, body)
		}
		return envelope.Data["message"]
	}

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		tb.Fatalf("%s: decode bare json response: %v; body: %s", stack.Name, err, body)
	}
	return payload["message"]
}

// Do sends a request through handler and returns the recorded response. It
// is exported so benchmark loops (not just Assert) can reuse the exact same
// request construction the correctness checks use.
func Do(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
