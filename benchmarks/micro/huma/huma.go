// Package huma is the bare Huma-over-Gin row of the BENCH-1 framework-tax
// microbenchmark matrix (issue #141): Huma's typed handlers, validation, and
// OpenAPI emission on top of Gin, without the Gombit runtime around it. The
// Gombit row (benchmarks/micro/gombit) registers the exact same routes
// (scenario.RegisterRoutes) through framework.App instead, so the delta
// between the two rows isolates the Gombit runtime's own cost.
package huma

import (
	humasdk "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	ginengine "github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/benchmarks/micro/scenario"
)

// NewServer returns a bare Huma-over-Gin router carrying the same four
// scenarios (plaintext, JSON, path parameter, validated POST) as the other
// rows.
func NewServer() *ginengine.Engine {
	ginengine.SetMode(ginengine.TestMode)
	router := ginengine.New()

	config := humasdk.DefaultConfig("Gombit Benchmark Micro (Huma+Gin)", "0.0.0")
	config.DocsPath = ""
	config.SchemasPath = ""

	api := humagin.New(router, config)
	scenario.RegisterRoutes(api)
	return router
}
