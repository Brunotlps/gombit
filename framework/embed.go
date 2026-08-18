package framework

import (
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// WithEmbeddedFrontend stores fsys and, when it contains index.html at its
// root, installs a Gin NoRoute handler after framework routes so unmatched
// GET paths serve the SPA. Huma /api/*, /openapi.json, /docs, and the
// probe routes still win because they are registered before NoRoute runs.
//
// If fsys has no index.html (the gombit new placeholder embed), NoRoute is
// not installed and unknown paths keep their current 404. Split deploy is
// the default (C5); embedding is opt-in via gombit build --embed.
func WithEmbeddedFrontend(fsys fs.FS) Option {
	return func(app *App) error {
		if fsys == nil {
			return errors.New("framework: nil embedded frontend")
		}
		app.embeddedFrontend = fsys
		return nil
	}
}

func mountEmbeddedFrontend(router *gin.Engine, fsys fs.FS) {
	if router == nil || fsys == nil {
		return
	}
	if !hasIndexHTML(fsys) {
		return
	}
	router.NoRoute(embeddedFrontendHandler(fsys))
}

func hasIndexHTML(fsys fs.FS) bool {
	info, err := fs.Stat(fsys, "index.html")
	return err == nil && info != nil && !info.IsDir()
}

func embeddedFrontendHandler(fsys fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		urlPath := path.Clean("/" + c.Request.URL.Path)
		if isReservedFrontendPath(urlPath) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		name := strings.TrimPrefix(urlPath, "/")
		if name != "" && name != "." && fs.ValidPath(name) {
			if serveEmbeddedFile(c, fsys, name) {
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

func isReservedFrontendPath(urlPath string) bool {
	switch urlPath {
	case "/livez", "/readyz", "/metrics",
		"/openapi.json", "/openapi.yaml",
		"/openapi-3.0.json", "/openapi-3.0.yaml",
		"/docs":
		return true
	}
	if urlPath == "/api" || strings.HasPrefix(urlPath, "/api/") {
		return true
	}
	if strings.HasPrefix(urlPath, "/docs/") {
		return true
	}
	return false
}

func serveEmbeddedFile(c *gin.Context, fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), rs)
		return true
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	c.Data(http.StatusOK, contentTypeFor(name), data)
	return true
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func contentTypeFor(name string) string {
	ctype := mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		return "application/octet-stream"
	}
	return ctype
}
