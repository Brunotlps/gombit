package main

import (
	"log"
	"net/http"

	"github.com/LAA-Software-Engineering/gombit/framework"
	"github.com/gin-gonic/gin"
)

// registerPingRoutes stands in for a feature package's routes.go,
// registered explicitly from main rather than discovered by the framework.
func registerPingRoutes(router *gin.Engine) {
	ping := router.Group("/ping")
	ping.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status":     "pong",
				"request_id": framework.GetRequestID(c),
				"trace_id":   framework.GetTraceID(c),
			},
		})
	})
}

// registerEchoRoutes stands in for a second, independent feature package.
// The default XSS middleware strips HTML tags from JSON string fields before
// this handler runs, so the echoed comment is the sanitized value.
func registerEchoRoutes(router *gin.Engine) {
	echo := router.Group("/echo")
	echo.POST("", func(c *gin.Context) {
		var body struct {
			Comment string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_json",
					"message": "request body must be JSON with a comment field",
				},
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"comment": body.Comment,
			},
		})
	})
}

func main() {
	app, err := framework.New()
	if err != nil {
		log.Fatal(err)
	}

	registerPingRoutes(app.Router())
	registerEchoRoutes(app.Router())

	if err := framework.Run(app); err != nil {
		log.Fatal(err)
	}
}
