package framework

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// mountAdminSPA registers explicit GET/HEAD routes for /admin and
// /admin/*filepath. Those win over Huma and over WithEmbeddedFrontend
// NoRoute. If fsys has no index.html, this is a no-op (same as M5-5).
func mountAdminSPA(router *gin.Engine, fsys fs.FS) {
	if router == nil || fsys == nil {
		return
	}
	if !hasIndexHTML(fsys) {
		return
	}
	handler := adminSPAHandler(fsys)
	router.GET("/admin", handler)
	router.HEAD("/admin", handler)
	router.GET("/admin/*filepath", handler)
	router.HEAD("/admin/*filepath", handler)
}

func adminSPAHandler(fsys fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		urlPath := path.Clean("/" + c.Request.URL.Path)
		if urlPath != "/admin" && !strings.HasPrefix(urlPath, "/admin/") {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		rel := strings.TrimPrefix(urlPath, "/admin")
		rel = strings.TrimPrefix(rel, "/")
		if rel != "" && rel != "." && fs.ValidPath(rel) {
			if serveEmbeddedFile(c, fsys, rel) {
				return
			}
		}

		if c.Request.Method != http.MethodGet {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		serveIndexHTML(c, fsys)
	}
}
