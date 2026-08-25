// Package gombitbench carries the Gombit-runtime row of the framework-tax
// benchmark matrix (issue #141) in its own package so it gets its own
// `go test` process, isolated from internal/contractspike.
//
// Constructing a framework.App calls contract.Install, which replaces Huma's
// process-global huma.NewError with Gombit's D10 error mapping, once, for
// the lifetime of the process (see contract.Install's doc comment). If the
// Gombit stack shared a test binary with internal/contractspike's
// TestCommittedOpenAPIJSONMatchesGeneratedDocument, whichever test ran first
// would decide the error schema baked into every OpenAPI document generated
// afterward in that process — including the widget routes' committed
// openapi.json, which is required to keep emitting Huma's default RFC 9457
// error schema (a documented M0 spike artifact, not Gombit's final error
// contract; see docs/spikes/M0-2_HUMA_GIN_SPIKE.md). Process isolation avoids
// relying on go test's file-ordering behavior to dodge that.
package gombitbench

import (
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/framework"
	"github.com/gombit-dev/gombit/internal/contractspike"
)

// NewBenchGombitApp returns a full framework.App carrying the same five
// framework-tax benchmark scenarios as internal/contractspike's net/http,
// Gin, and Huma+Gin stacks, registered through the runtime's Huma API
// (app.API()) so the benchmark measures the whole Gombit runtime —
// request-id, security headers, XSS sanitization, D10 error mapping — not
// just bare Huma+Gin.
func NewBenchGombitApp() *framework.App {
	cfg := config.DefaultFor(config.EnvironmentProduction)
	cfg.HTTP.Addr = "127.0.0.1:0"

	app, err := framework.New(framework.WithConfig(cfg))
	if err != nil {
		panic("gombitbench: build bench Gombit app: " + err.Error())
	}

	contractspike.RegisterBenchRoutes(app.API())
	return app
}
