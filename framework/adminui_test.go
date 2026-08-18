package framework

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestMountAdminSPASkipsWithoutIndex(t *testing.T) {
	previous := gin.Mode()
	t.Cleanup(func() { gin.SetMode(previous) })
	gin.SetMode(gin.TestMode)

	router := gin.New()
	mountAdminSPA(router, fstest.MapFS{".keep": {Data: []byte("")}})

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /admin/ status = %d, want %d without index.html", rec.Code, http.StatusNotFound)
	}
}

func TestEmbeddedFrontendDoesNotTakeAdminPath(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))

	rec := serveEmbed(t, app, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK || rec.Body.String() != embedIndexBody {
		t.Fatalf("GET / = %d %q, want app SPA index", rec.Code, rec.Body.String())
	}

	rec = serveEmbed(t, app, http.MethodGet, "/admin/", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /admin/ status = %d, want %d (reserved; JWT/no-cookie app)", rec.Code, http.StatusNotFound)
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /admin/ served the application SPA index")
	}

	rec = serveEmbed(t, app, http.MethodGet, "/admin/login", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /admin/login status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAdminSPAHandlerRejectsDotDot(t *testing.T) {
	previous := gin.Mode()
	t.Cleanup(func() { gin.SetMode(previous) })
	gin.SetMode(gin.TestMode)

	fsys := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><div id=\"root\">admin</div>")},
		"assets/app.js": {Data: []byte("console.log('admin')\n")},
	}
	router := gin.New()
	mountAdminSPA(router, fsys)

	req := httptest.NewRequest(http.MethodGet, "/admin/assets/../assets/app.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cleaned asset path status = %d; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "console.log('admin')\n" {
		t.Fatalf("cleaned asset path served %q", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/foo/../../etc/passwd", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("escaped path status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminSPAHandlerServesIndexAndAsset(t *testing.T) {
	previous := gin.Mode()
	t.Cleanup(func() { gin.SetMode(previous) })
	gin.SetMode(gin.TestMode)

	fsys := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><div id=\"root\">admin</div>")},
		"assets/app.js": {Data: []byte("console.log('admin')\n")},
	}
	router := gin.New()
	mountAdminSPA(router, fsys)

	for _, path := range []string{"/admin", "/admin/", "/admin/login", "/admin/widgets"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d; body=%s", path, rec.Code, http.StatusOK, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("GET %s Content-Type = %q, want text/html", path, ct)
		}
		if !strings.Contains(rec.Body.String(), "id=\"root\"") {
			t.Fatalf("GET %s body = %q, want index.html", path, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/assets/app.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET asset status = %d; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "console.log('admin')\n" {
		t.Fatalf("GET asset body = %q", rec.Body.String())
	}
}
