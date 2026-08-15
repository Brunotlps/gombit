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
func registerEchoRoutes(router *gin.Engine) {
	echo := router.Group("/echo")
	echo.POST("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
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
