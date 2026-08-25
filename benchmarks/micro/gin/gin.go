// Package gin is the idiomatic plain-Gin row of the BENCH-1 framework-tax
// microbenchmark matrix (issue #141): Gin routing and binding-tag
// validation, no Huma, no Gombit framework.
package gin

import (
	"net/http"

	ginengine "github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/benchmarks/micro/scenario"
)

// createUserRequest uses Gin's own binding-tag validation
// (go-playground/validator under the hood) instead of Huma's JSON-schema
// validation — the idiomatic Gin equivalent of scenario.CreateBenchUserBody.
type createUserRequest struct {
	Name  string `json:"name" binding:"required,max=80"`
	Email string `json:"email" binding:"required,email"`
}

// NewRouter returns the idiomatic plain-Gin baseline carrying the same four
// scenarios (plaintext, JSON, path parameter, validated POST) as the other
// rows.
func NewRouter() *ginengine.Engine {
	ginengine.SetMode(ginengine.TestMode)
	router := ginengine.New()

	router.GET("/plaintext", func(c *ginengine.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	router.GET("/json", func(c *ginengine.Context) {
		c.JSON(http.StatusOK, ginengine.H{"message": "Hello, World!"})
	})

	router.GET("/users/:id", func(c *ginengine.Context) {
		c.JSON(http.StatusOK, scenario.BenchUser{ID: c.Param("id"), Name: "Ada Lovelace"})
	})

	router.POST("/users", func(c *ginengine.Context) {
		var body createUserRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusUnprocessableEntity, ginengine.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, scenario.BenchUser{ID: "user-1", Name: body.Name, Email: body.Email})
	})

	return router
}
