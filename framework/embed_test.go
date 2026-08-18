package framework

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

const embedIndexBody = "<!doctype html><title>spa</title>index"

func TestEmbeddedFrontendServesIndexAndAssets(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))

	for _, path := range []string{"/", "/login", "/products/new"} {
		rec := serveEmbed(t, app, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d; body=%s", path, rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Body.String(); got != embedIndexBody {
			t.Fatalf("GET %s body = %q, want index.html", path, got)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("GET %s Content-Type = %q, want text/html", path, ct)
		}
	}

	rec := serveEmbed(t, app, http.MethodGet, "/assets/app.js", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/app.js status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Body.String(); got != "console.log('app')\n" {
		t.Fatalf("GET /assets/app.js body = %q, want asset bytes", got)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") && !strings.Contains(ct, "ecmascript") {
		t.Fatalf("GET /assets/app.js Content-Type = %q, want javascript", ct)
	}
}

func TestEmbeddedFrontendDoesNotTakeOverAPIOrProbes(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))

	type pingOutput struct {
		Body contract.Data[map[string]string]
	}
	prefix := app.Config().API.Prefix
	huma.Register(app.API(), huma.Operation{
		OperationID: "embed-ping",
		Method:      http.MethodGet,
		Path:        prefix + "/ping",
		Summary:     "Ping",
	}, func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return &pingOutput{Body: contract.Data[map[string]string]{Data: map[string]string{"status": "ok"}}}, nil
	})

	rec := serveEmbed(t, app, http.MethodGet, prefix+"/ping", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d; body=%s", prefix+"/ping", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatalf("GET %s served index.html, want API JSON: %s", prefix+"/ping", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) && !strings.Contains(rec.Body.String(), `"status": "ok"`) {
		t.Fatalf("GET %s body = %q, want API payload", prefix+"/ping", rec.Body.String())
	}

	rec = serveEmbed(t, app, http.MethodGet, "/readyz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /readyz status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /readyz served index.html, want probe JSON")
	}
	if !strings.Contains(rec.Body.String(), `"status"`) {
		t.Fatalf("GET /readyz body = %q, want probe payload", rec.Body.String())
	}

	rec = serveEmbed(t, app, http.MethodGet, "/livez", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /livez status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /livez served index.html")
	}

	rec = serveEmbed(t, app, http.MethodGet, "/openapi.json", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /openapi.json served index.html")
	}

	rec = serveEmbed(t, app, http.MethodGet, prefix+"/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET %s status = %d, want %d; body=%s", prefix+"/missing", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatalf("GET %s served index.html, want API 404", prefix+"/missing")
	}

	rec = serveEmbed(t, app, http.MethodGet, "/docs", nil)
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /docs served index.html, want Huma swagger (docs on)")
	}

	rec = serveEmbed(t, app, http.MethodGet, "/metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /metrics served index.html, want metrics text")
	}
	if !strings.Contains(rec.Body.String(), "gombit_http") {
		t.Fatalf("GET /metrics body = %q, want probe/metrics payload", rec.Body.String())
	}
}

func TestEmbeddedFrontendDocsOffIsNotSPA(t *testing.T) {
	previousMode := gin.Mode()
	t.Cleanup(func() {
		gin.SetMode(previousMode)
	})

	cfg := config.DefaultFor(config.EnvironmentProduction)
	cfg.HTTP.Addr = "127.0.0.1:0"
	if cfg.API.DocsEnabled {
		t.Fatal("precondition: DefaultFor(production) DocsEnabled = true, want false")
	}
	app := newTestApp(t, WithConfig(cfg), WithEmbeddedFrontend(spaFixtureFS()))

	rec := serveEmbed(t, app, http.MethodGet, "/docs", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /docs status = %d, want %d with docs off + embed", rec.Code, http.StatusNotFound)
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("GET /docs served index.html with DocsEnabled=false")
	}

	rec = serveEmbed(t, app, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != embedIndexBody {
		t.Fatalf("GET / body = %q, want SPA index.html", rec.Body.String())
	}
}

func TestEmbeddedFrontendIndexUsesSPACSP(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))

	rec := serveEmbed(t, app, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "fonts.googleapis.com") {
		t.Fatalf("GET / CSP = %q, want fonts.googleapis.com", csp)
	}
	style := cspDirective(csp, "style-src")
	if style == "" || !strings.Contains(style, "'unsafe-inline'") {
		t.Fatalf("GET / CSP = %q, want style-src with 'unsafe-inline'", csp)
	}
	script := cspDirective(csp, "script-src")
	if script == "" {
		t.Fatalf("GET / CSP = %q, want script-src 'self'", csp)
	}
	if strings.Contains(script, "'unsafe-inline'") {
		t.Fatalf("GET / CSP = %q, must not put 'unsafe-inline' on script-src", csp)
	}

	rec = serveEmbed(t, app, http.MethodGet, "/readyz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /readyz status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	readyCSP := rec.Header().Get("Content-Security-Policy")
	if readyCSP != "default-src 'self'" {
		t.Fatalf("GET /readyz CSP = %q, want default-src 'self'", readyCSP)
	}
	if strings.Contains(readyCSP, "fonts.googleapis.com") {
		t.Fatalf("GET /readyz CSP = %q, must not get SPA font hosts", readyCSP)
	}
}

func TestEmbeddedFrontendPostUnmatchedIsNotIndex(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))

	rec := serveEmbed(t, app, http.MethodPost, "/login", strings.NewReader(`{}`))
	if rec.Code == http.StatusOK && rec.Body.String() == embedIndexBody {
		t.Fatal("POST /login served index.html")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /login status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatalf("POST /login body = %q, want not index.html", rec.Body.String())
	}
}

func TestEmbeddedFrontendWithoutIndexDoesNotTakeOver(t *testing.T) {
	fsys := fstest.MapFS{
		".keep": {Data: []byte("")},
	}
	app := newTestApp(t, WithEmbeddedFrontend(fsys))

	rec := serveEmbed(t, app, http.MethodGet, "/login", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /login status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), embedIndexBody) {
		t.Fatal("placeholder embed served index.html")
	}

	rec = serveEmbed(t, app, http.MethodGet, "/", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET / status = %d, want %d without index.html", rec.Code, http.StatusNotFound)
	}
}

func TestEmbeddedFrontendRejectsNilFS(t *testing.T) {
	_, err := New(WithEmbeddedFrontend(nil))
	if err == nil {
		t.Fatal("New(WithEmbeddedFrontend(nil)) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "nil embedded frontend") {
		t.Fatalf("error = %q, want nil embedded frontend", err)
	}
}

func TestEmbeddedFrontendLeavesGinEscapeHatch(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))
	app.Router().GET("/custom", func(c *gin.Context) {
		c.String(http.StatusOK, "escape")
	})

	rec := serveEmbed(t, app, http.MethodGet, "/custom", nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "escape" {
		t.Fatalf("GET /custom = %d %q, want 200 escape", rec.Code, rec.Body.String())
	}
}

func TestEmbeddedFrontendRejectsDotDot(t *testing.T) {
	app := newTestApp(t, WithEmbeddedFrontend(spaFixtureFS()))
	rec := serveEmbed(t, app, http.MethodGet, "/assets/../assets/app.js", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("cleaned asset path status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "console.log('app')\n" {
		t.Fatalf("cleaned asset path served %q, want asset bytes", rec.Body.String())
	}

	rec = serveEmbed(t, app, http.MethodGet, "/../index.html", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /../index.html status = %d", rec.Code)
	}
	if rec.Body.String() != embedIndexBody {
		t.Fatalf("traversal escaped embed FS: %q", rec.Body.String())
	}
}

func cspDirective(csp, name string) string {
	for _, part := range strings.Split(csp, ";") {
		part = strings.TrimSpace(part)
		if part == name || strings.HasPrefix(part, name+" ") {
			return part
		}
	}
	return ""
}

func spaFixtureFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte(embedIndexBody)},
		"assets/app.js": {Data: []byte("console.log('app')\n")},
	}
}

func serveEmbed(t *testing.T, app *App, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if body != nil && method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, req)
	return rec
}
