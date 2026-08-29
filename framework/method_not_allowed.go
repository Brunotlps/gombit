package framework

import (
	"github.com/gin-gonic/gin"

	"github.com/gombit-dev/gombit/contract"
)

// enableMethodNotAllowed makes Gin distinguish "no such route" (404, handled by
// the SPA NoRoute fallback) from "route exists, method not supported" (405).
// Without HandleMethodNotAllowed, a PUT/PATCH/DELETE to a GET/POST-only path
// (e.g. /api/v1/engines/1) falls through to NoRoute and returns 404 — clients
// can't tell a missing resource from an unsupported method. See #225.
//
// Gin itself already generates the RFC 7231 Allow header from its own method
// trees before dispatching to the NoMethod handler (gin.go handleHTTPRequest),
// so we must not overwrite it — the NoMethod handler only supplies the D10 405
// envelope body. Global middleware (request_id, etc.) still runs before it, so
// the envelope carries request_id.
func enableMethodNotAllowed(router *gin.Engine) {
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		env := contract.WithContext(c.Request.Context(), contract.MethodNotAllowed(""))
		c.AbortWithStatusJSON(env.GetStatus(), env)
	})
}
