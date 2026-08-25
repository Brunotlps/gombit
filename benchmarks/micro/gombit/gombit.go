// Package gombit is the Gombit-runtime row of the BENCH-1 framework-tax
// microbenchmark matrix (issue #141): the same four scenarios
// (scenario.RegisterRoutes) as benchmarks/micro/huma, but registered through
// a real framework.App instead of bare Huma+Gin, so the delta between the
// two rows isolates the cost of the Gombit runtime itself — request-id,
// security headers, XSS sanitization, D10 error mapping — on top of Huma.
//
// This package must stay in its own directory/package, separate from the
// other three rows: constructing a framework.App calls contract.Install,
// which replaces Huma's process-global huma.NewError with Gombit's D10
// error mapping, once, for the lifetime of the process (see
// contract.Install's doc comment). Since `go test` runs one process per
// package, keeping this row in its own package is what makes that safe
// rather than order-dependent.
package gombit

import (
	"github.com/gombit-dev/gombit/benchmarks/micro/scenario"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/framework"
)

// NewApp returns a full framework.App carrying the same four scenarios as
// the other rows, registered through the runtime's Huma API (app.API()).
func NewApp() *framework.App {
	cfg := config.DefaultFor(config.EnvironmentProduction)
	cfg.HTTP.Addr = "127.0.0.1:0"

	app, err := framework.New(framework.WithConfig(cfg))
	if err != nil {
		panic("benchmarks/micro/gombit: build bench app: " + err.Error())
	}

	scenario.RegisterRoutes(app.API())
	return app
}
