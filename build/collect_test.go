package build

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/framework"
	"github.com/gin-gonic/gin"
)

func TestCollectStaticCopiesIndexAndAssets(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "frontend", "dist")
	dst := filepath.Join(root, "internal", "web", "static")
	writeFile(t, filepath.Join(src, "index.html"), "<html>spa</html>")
	writeFile(t, filepath.Join(src, "assets", "app.js"), "console.log(1)")
	writeFile(t, filepath.Join(dst, keepName), "")
	writeFile(t, filepath.Join(dst, "old.js"), "stale")
	writeFile(t, filepath.Join(root, "internal", "web", "embed.go"), "package web\n")

	if err := CollectStatic(src, dst); err != nil {
		t.Fatalf("CollectStatic() error = %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "index.html")); got != "<html>spa</html>" {
		t.Fatalf("index.html = %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "assets", "app.js")); got != "console.log(1)" {
		t.Fatalf("app.js = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "old.js")); !os.IsNotExist(err) {
		t.Fatal("previous asset old.js was not replaced")
	}
	if _, err := os.Stat(filepath.Join(dst, keepName)); err != nil {
		t.Fatalf(".keep was deleted: %v", err)
	}
	if got := readFile(t, filepath.Join(root, "internal", "web", "embed.go")); got != "package web\n" {
		t.Fatalf("embed.go was modified: %q", got)
	}
}

func TestCollectStaticRequiresIndexHTML(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "frontend", "dist")
	dst := filepath.Join(root, "internal", "web", "static")
	writeFile(t, filepath.Join(src, "assets", "app.js"), "js")
	if err := os.MkdirAll(dst, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err := CollectStatic(src, dst)
	if err == nil {
		t.Fatal("CollectStatic() error = nil, want missing index.html")
	}
	if !strings.Contains(err.Error(), "index.html") {
		t.Fatalf("CollectStatic() error = %q, want index.html", err)
	}
}

func TestCollectStaticSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "dist")
	dst := filepath.Join(root, "static")
	secret := filepath.Join(root, "secret.txt")
	writeFile(t, filepath.Join(src, "index.html"), "<html>ok</html>")
	writeFile(t, secret, "outside-secret")
	if err := os.Symlink(secret, filepath.Join(src, "leak.txt")); err != nil {
		t.Skipf("symlink: %v", err)
	}

	if err := CollectStatic(src, dst); err != nil {
		t.Fatalf("CollectStatic() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "leak.txt")); !os.IsNotExist(err) {
		t.Fatal("symlink leak.txt was copied into static")
	}
	if got := readFile(t, filepath.Join(dst, "index.html")); got != "<html>ok</html>" {
		t.Fatalf("index.html = %q", got)
	}
}

func TestCollectStaticDirFSServesAPIAndSPA(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "dist")
	dst := filepath.Join(root, "static")
	writeFile(t, filepath.Join(src, "index.html"), "<!doctype html>collected")
	writeFile(t, filepath.Join(src, "assets", "app.js"), "asset-bytes")
	if err := CollectStatic(src, dst); err != nil {
		t.Fatalf("CollectStatic() error = %v", err)
	}

	app := newCollectApp(t, os.DirFS(dst))
	app.Router().GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	if rec := get("/"); rec.Code != http.StatusOK || rec.Body.String() != "<!doctype html>collected" {
		t.Fatalf("GET / = %d %q", rec.Code, rec.Body.String())
	}
	if rec := get("/login"); rec.Code != http.StatusOK || rec.Body.String() != "<!doctype html>collected" {
		t.Fatalf("GET /login = %d %q", rec.Code, rec.Body.String())
	}
	if rec := get("/assets/app.js"); rec.Code != http.StatusOK || rec.Body.String() != "asset-bytes" {
		t.Fatalf("GET /assets/app.js = %d %q", rec.Code, rec.Body.String())
	}
	if rec := get("/api/v1/ping"); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "collected") {
		t.Fatalf("GET /api/v1/ping = %d %q, want API JSON", rec.Code, rec.Body.String())
	}
}

func newCollectApp(t *testing.T, fsys fs.FS) *framework.App {
	t.Helper()
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.Addr = "127.0.0.1:0"
	app, err := framework.New(framework.WithConfig(cfg), framework.WithEmbeddedFrontend(fsys))
	if err != nil {
		t.Fatalf("framework.New: %v", err)
	}
	return app
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- test fixture path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
