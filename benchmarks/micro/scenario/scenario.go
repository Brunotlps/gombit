// Package scenario defines the five request/response scenarios shared by
// every row of the BENCH-1 framework-tax microbenchmark matrix (issue #141
// "1. Go abstraction-overhead microbenchmarks"): net/http -> Gin -> Huma+Gin
// -> Gombit, in benchmarks/micro/{nethttp,gin,huma,gombit}. Each stack
// implements the scenarios idiomatically, but they share this package's
// resource types so the comparison stays apples-to-apples: every stack
// serializes the same BenchUser shape.
//
// This package intentionally does not depend on internal/contractspike. The
// M0-2 spike that lives there (docs/spikes/M0-2_HUMA_GIN_SPIKE.md) is
// preserved as a historical, standalone artifact; this matrix supersedes it
// as the ongoing benchmark but does not extend its code.
package scenario

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// SuccessEnvelope is the D10 success response shape. It is a Gombit
// convention, not something Huma imposes by default, so only
// RegisterEnvelopedRoutes (the Gombit row) uses it — RegisterRoutes (the
// bare Huma+Gin row) does not. Wrapping both rows in it would bill Gombit's
// envelope cost to the Huma+Gin baseline too and understate the delta
// between them, which is the whole point of comparing those two rows.
type SuccessEnvelope[T any] struct {
	Data T `json:"data"`
}

// BenchUser is the JSON resource shared by the path-parameter and POST
// scenarios. net/http, Gin, and bare Huma+Gin return it bare; only Gombit
// wraps it in SuccessEnvelope, matching what each stack would actually do
// left to its own defaults.
type BenchUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// CreateBenchUserBody is the validated POST /users request body. Huma and
// Gombit enforce the pattern tag at request-validation time (not just in the
// generated OpenAPI schema), so the invalid-POST scenario exercises real
// validation work; net/http and Gin reuse this struct's JSON shape for their
// own idiomatic (manual / binding-tag) validation.
type CreateBenchUserBody struct {
	Name  string `json:"name" minLength:"1" maxLength:"80" example:"Ada Lovelace" doc:"Human-readable user name"`
	Email string `json:"email" pattern:"^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$" example:"ada@example.com" doc:"Contact email address"`
}

type plaintextOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

type getBenchUserInput struct {
	ID string `path:"id"`
}

type createBenchUserInput struct {
	Body CreateBenchUserBody
}

type jsonMessageOutput struct {
	Body map[string]string
}

type getBenchUserOutput struct {
	Body BenchUser
}

type createBenchUserOutput struct {
	Body BenchUser
}

type envelopedJSONMessageOutput struct {
	Body SuccessEnvelope[map[string]string]
}

type envelopedGetBenchUserOutput struct {
	Body SuccessEnvelope[BenchUser]
}

type envelopedCreateBenchUserOutput struct {
	Body SuccessEnvelope[BenchUser]
}

// plaintextHandler is shared by both registrations: the plaintext scenario
// has no JSON body, so there is no envelope question for it either way.
func plaintextHandler(ctx context.Context, input *struct{}) (*plaintextOutput, error) {
	return &plaintextOutput{
		ContentType: "text/plain; charset=utf-8",
		Body:        []byte("Hello, World!"),
	}, nil
}

// RegisterRoutes registers the four Huma-typed scenario routes (plaintext,
// JSON, path parameter, validated POST) onto api without the D10 envelope —
// what a bare Huma+Gin app returns left to Huma's own defaults. Used by the
// bare Huma+Gin row (benchmarks/micro/huma).
func RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "bench-plaintext",
		Method:      http.MethodGet,
		Path:        "/plaintext",
		Summary:     "Plaintext benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, plaintextHandler)

	huma.Register(api, huma.Operation{
		OperationID: "bench-json",
		Method:      http.MethodGet,
		Path:        "/json",
		Summary:     "JSON benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *struct{}) (*jsonMessageOutput, error) {
		return &jsonMessageOutput{Body: map[string]string{"message": "Hello, World!"}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-get-user",
		Method:      http.MethodGet,
		Path:        "/users/{id}",
		Summary:     "Path parameter benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *getBenchUserInput) (*getBenchUserOutput, error) {
		return &getBenchUserOutput{Body: BenchUser{ID: input.ID, Name: "Ada Lovelace"}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-create-user",
		Method:      http.MethodPost,
		Path:        "/users",
		Summary:     "Validated POST benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *createBenchUserInput) (*createBenchUserOutput, error) {
		return &createBenchUserOutput{
			Body: BenchUser{ID: "user-1", Name: input.Body.Name, Email: input.Body.Email},
		}, nil
	})
}

// RegisterEnvelopedRoutes registers the same four scenarios as
// RegisterRoutes, but wrapped in the D10 SuccessEnvelope, matching what a
// real Gombit app returns. Used by the Gombit row (benchmarks/micro/gombit)
// so the delta against RegisterRoutes (bare Huma+Gin) isolates the Gombit
// runtime's own cost — request-id, security headers, XSS sanitization, D10
// error mapping, and this envelope — rather than conflating it with a
// response-shape choice bare Huma+Gin never made.
func RegisterEnvelopedRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "bench-plaintext",
		Method:      http.MethodGet,
		Path:        "/plaintext",
		Summary:     "Plaintext benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, plaintextHandler)

	huma.Register(api, huma.Operation{
		OperationID: "bench-json",
		Method:      http.MethodGet,
		Path:        "/json",
		Summary:     "JSON benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *struct{}) (*envelopedJSONMessageOutput, error) {
		return &envelopedJSONMessageOutput{
			Body: SuccessEnvelope[map[string]string]{Data: map[string]string{"message": "Hello, World!"}},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-get-user",
		Method:      http.MethodGet,
		Path:        "/users/{id}",
		Summary:     "Path parameter benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *getBenchUserInput) (*envelopedGetBenchUserOutput, error) {
		return &envelopedGetBenchUserOutput{
			Body: SuccessEnvelope[BenchUser]{Data: BenchUser{ID: input.ID, Name: "Ada Lovelace"}},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-create-user",
		Method:      http.MethodPost,
		Path:        "/users",
		Summary:     "Validated POST benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *createBenchUserInput) (*envelopedCreateBenchUserOutput, error) {
		return &envelopedCreateBenchUserOutput{
			Body: SuccessEnvelope[BenchUser]{
				Data: BenchUser{ID: "user-1", Name: input.Body.Name, Email: input.Body.Email},
			},
		}, nil
	})
}

// Fixture request bodies shared by every stack's POST /users scenarios.
const (
	ValidCreateUserBody   = `{"name":"Ada Lovelace","email":"ada@example.com"}`
	InvalidCreateUserBody = `{"name":"","email":"not-an-email"}`
)
