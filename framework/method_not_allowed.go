package framework

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gombit-dev/gombit/contract"
)

// enableMethodNotAllowed makes Gin distinguish "no such route" (404, handled by
// the SPA NoRoute fallback) from "route exists, method not supported" (405).
// Without HandleMethodNotAllowed, a PUT/PATCH/DELETE to a GET/POST-only path
// (e.g. /api/v1/engines/1) falls through to NoRoute and returns 404 — clients
// can't tell a missing resource from an unsupported method. See #225.
//
// The NoMethod handler emits the D10 405 envelope and an Allow header listing
// the methods the concrete path does support, computed from the router's own
// route table. Global middleware (request_id, etc.) still runs before it, so
// the envelope carries request_id.
func enableMethodNotAllowed(router *gin.Engine) {
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		env := contract.WithContext(c.Request.Context(), contract.MethodNotAllowed(""))
		if allow := allowedMethodsFor(router, c.Request.URL.Path); allow != "" {
			c.Header("Allow", allow)
		}
		c.AbortWithStatusJSON(env.GetStatus(), env)
	})
}

// allowedMethodsFor returns a sorted, comma-separated Allow header value listing
// every method registered for a route pattern matching reqPath. It returns ""
// when nothing matches (Gin only invokes NoMethod when at least one method's
// tree matched, so this is a defensive fallback).
func allowedMethodsFor(router *gin.Engine, reqPath string) string {
	seen := map[string]struct{}{}
	var methods []string
	for _, r := range router.Routes() {
		if !routePatternMatches(r.Path, reqPath) {
			continue
		}
		if _, ok := seen[r.Method]; ok {
			continue
		}
		seen[r.Method] = struct{}{}
		methods = append(methods, r.Method)
	}
	sort.Strings(methods)
	return strings.Join(methods, ", ")
}

// routePatternMatches reports whether a Gin route pattern (with :param and
// *catchAll segments) matches a concrete request path.
func routePatternMatches(pattern, reqPath string) bool {
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	rp := strings.Split(strings.Trim(reqPath, "/"), "/")
	for i, seg := range pp {
		if strings.HasPrefix(seg, "*") {
			// Catch-all consumes the remainder of the path.
			return true
		}
		if i >= len(rp) {
			return false
		}
		if strings.HasPrefix(seg, ":") {
			if rp[i] == "" {
				return false
			}
			continue
		}
		if seg != rp[i] {
			return false
		}
	}
	return len(pp) == len(rp)
}
