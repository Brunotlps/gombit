package framework

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/gombit-dev/gombit/contract"
)

// TestMethodNotAllowedReturns405WithAllow verifies #225 item 1: an unsupported
// method on a known path returns 405 (not 404) with an Allow header and the D10
// envelope, so clients can distinguish "no such resource" from "method not
// supported".
func TestMethodNotAllowedReturns405WithAllow(t *testing.T) {
	app := newTestApp(t)
	prefix := app.Config().API.Prefix

	huma.Register(app.API(), huma.Operation{
		OperationID: "get-engine",
		Method:      http.MethodGet,
		Path:        prefix + "/engines/{id}",
	}, func(ctx context.Context, input *struct {
		ID int `path:"id"`
	}) (*struct {
		Body contract.Data[string]
	}, error) {
		return &struct{ Body contract.Data[string] }{Body: contract.Data[string]{Data: "ok"}}, nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, prefix+"/engines/1", nil)
	req.Header.Set(RequestIDHeader, "req-405")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	allow := rec.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) {
		t.Fatalf("Allow = %q, want it to contain GET", allow)
	}
	if strings.Contains(allow, http.MethodDelete) {
		t.Fatalf("Allow = %q, must not list the unsupported method DELETE", allow)
	}

	var body contract.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, rec.Body.String())
	}
	if body.Body.Code != contract.CodeMethodNotAllowed {
		t.Fatalf("code = %q, want %q", body.Body.Code, contract.CodeMethodNotAllowed)
	}
	if body.Body.RequestID != "req-405" {
		t.Fatalf("request_id = %q, want req-405", body.Body.RequestID)
	}
}

// TestUnknownPathStillReturns404 guards that enabling HandleMethodNotAllowed
// does not turn a genuinely missing path into a 405: a path with no registered
// route at all still falls through (404), leaving the SPA NoRoute fallback and
// existing 404 semantics intact.
func TestUnknownPathStillReturns404(t *testing.T) {
	app := newTestApp(t)
	prefix := app.Config().API.Prefix

	huma.Register(app.API(), huma.Operation{
		OperationID: "get-engine-404case",
		Method:      http.MethodGet,
		Path:        prefix + "/engines/{id}",
	}, func(ctx context.Context, input *struct {
		ID int `path:"id"`
	}) (*struct {
		Body contract.Data[string]
	}, error) {
		return &struct{ Body contract.Data[string] }{Body: contract.Data[string]{Data: "ok"}}, nil
	})

	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, prefix+"/nonexistent", nil))
	if rec.Code == http.StatusMethodNotAllowed {
		t.Fatalf("unknown path returned 405, want 404-style fallback; body: %s", rec.Body.String())
	}
}

func TestRoutePatternMatches(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/api/v1/engines/:id", "/api/v1/engines/1", true},
		{"/api/v1/engines/:id", "/api/v1/engines", false},
		{"/api/v1/engines", "/api/v1/engines", true},
		{"/api/v1/engines/:id", "/api/v1/engines/1/extra", false},
		{"/assets/*filepath", "/assets/js/app.js", true},
		{"/api/v1/engines/:id", "/api/v1/widgets/1", false},
	}
	for _, tc := range cases {
		if got := routePatternMatches(tc.pattern, tc.path); got != tc.want {
			t.Errorf("routePatternMatches(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}
