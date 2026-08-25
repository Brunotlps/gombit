package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/auth"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/contract"
	"github.com/gombit-dev/gombit/framework"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// cookieJar is a minimal same-name-wins cookie store for exercising the
// cookie-mode auth surface without pulling in net/http/cookiejar (which
// requires real URLs rather than httptest's in-memory router).
type cookieJar struct {
	cookies map[string]*http.Cookie
}

func newCookieJar() *cookieJar {
	return &cookieJar{cookies: map[string]*http.Cookie{}}
}

func (j *cookieJar) update(rec *httptest.ResponseRecorder) {
	for _, c := range rec.Result().Cookies() {
		j.cookies[c.Name] = c
	}
}

func (j *cookieJar) attach(req *http.Request) {
	for _, c := range j.cookies {
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value}) //nolint:gosec // G124: request Cookie header only carries name/value.
	}
}

func (j *cookieJar) value(name string) string {
	if c, ok := j.cookies[name]; ok {
		return c.Value
	}
	return ""
}

func newCookieAuthApp(t *testing.T) *framework.App {
	t.Helper()
	db := openSQLite(t)
	if err := auth.Migrate(db.DB); err != nil {
		t.Fatalf("auth.Migrate() error = %v", err)
	}
	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.Auth.JWTSecret = testJWTSecret
	cfg.Auth.BcryptCost = bcrypt.MinCost
	cfg.Auth.AccessTokenTTL = time.Minute
	cfg.Auth.RefreshTokenTTL = time.Hour
	cfg.Auth.Mode = config.AuthModeCookie
	app, err := framework.New(
		framework.WithConfig(cfg),
		framework.WithDatabase(db),
		framework.WithLogger(zap.NewNop()),
	)
	if err != nil {
		t.Fatalf("framework.New() error = %v", err)
	}
	return app
}

func doRequest(app *framework.App, jar *cookieJar, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if jar != nil {
		jar.attach(req)
		if csrf := jar.value(auth.CSRFCookieName); csrf != "" {
			req.Header.Set(auth.CSRFHeaderName, csrf)
		}
	}
	app.Router().ServeHTTP(rec, req)
	if jar != nil {
		jar.update(rec)
	}
	return rec
}

func fetchCSRF(t *testing.T, app *framework.App) *cookieJar {
	t.Helper()
	jar := newCookieJar()
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/auth/csrf", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /auth/csrf status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if jar.value(auth.CSRFCookieName) == "" {
		t.Fatalf("GET /auth/csrf did not set %s cookie; headers: %v", auth.CSRFCookieName, rec.Header())
	}
	return jar
}

func registerCookieUser(t *testing.T, app *framework.App, jar *cookieJar, email, password string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := doRequest(app, jar, http.MethodPost, "/api/v1/auth/register", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("register status = %d; body: %s", rec.Code, rec.Body.String())
	}
}

func loginCookieUser(t *testing.T, app *framework.App, jar *cookieJar, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return doRequest(app, jar, http.MethodPost, "/api/v1/auth/login", string(body))
}

func getMeCookie(app *framework.App, jar *cookieJar) int {
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/me", "")
	return rec.Code
}

func TestCookieE2ELoginRefreshLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieAuthApp(t)

	jar := fetchCSRF(t, app)
	registerCookieUser(t, app, jar, "cookie-e2e@example.com", testPassword)

	loginRec := loginCookieUser(t, app, jar, "cookie-e2e@example.com", testPassword)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d; body: %s", loginRec.Code, loginRec.Body.String())
	}
	if jar.value(auth.AccessCookieName) == "" || jar.value(auth.RefreshCookieName) == "" {
		t.Fatalf("login did not set session cookies; jar = %+v", jar.cookies)
	}

	if status := getMeCookie(app, jar); status != http.StatusOK {
		t.Fatalf("GET /me after login status = %d, want 200", status)
	}

	oldAccess := jar.value(auth.AccessCookieName)
	oldRefresh := jar.value(auth.RefreshCookieName)
	refreshRec := doRequest(app, jar, http.MethodPost, "/api/v1/auth/refresh", "")
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d; body: %s", refreshRec.Code, refreshRec.Body.String())
	}
	if jar.value(auth.AccessCookieName) == oldAccess {
		t.Fatal("access cookie was not rotated")
	}
	if jar.value(auth.RefreshCookieName) == oldRefresh {
		// refresh token rotation always changes the value; this branch
		// should not be reached, but keep the assertion explicit.
		t.Fatal("refresh cookie unexpectedly unchanged")
	}

	if status := getMeCookie(app, jar); status != http.StatusOK {
		t.Fatalf("GET /me after refresh status = %d, want 200", status)
	}

	logoutRec := doRequest(app, jar, http.MethodPost, "/api/v1/auth/logout", "")
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d; body: %s", logoutRec.Code, logoutRec.Body.String())
	}
	if jar.value(auth.AccessCookieName) != "" {
		t.Fatal("logout did not clear access cookie")
	}

	if status := getMeCookie(app, jar); status != http.StatusUnauthorized {
		t.Fatalf("GET /me after logout status = %d, want 401", status)
	}
}

func TestCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		sameSite     config.CookieSameSite
		secure       bool
		wantSameSite http.SameSite
	}{
		{name: "lax+secure", sameSite: config.CookieSameSiteLax, secure: true, wantSameSite: http.SameSiteLaxMode},
		{name: "strict+secure", sameSite: config.CookieSameSiteStrict, secure: true, wantSameSite: http.SameSiteStrictMode},
		{name: "lax+insecure", sameSite: config.CookieSameSiteLax, secure: false, wantSameSite: http.SameSiteLaxMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openSQLite(t)
			if err := auth.Migrate(db.DB); err != nil {
				t.Fatalf("auth.Migrate() error = %v", err)
			}
			cfg := config.DefaultFor(config.EnvironmentTest)
			cfg.HTTP.Addr = "127.0.0.1:0"
			cfg.Auth.JWTSecret = testJWTSecret
			cfg.Auth.BcryptCost = bcrypt.MinCost
			cfg.Auth.AccessTokenTTL = time.Minute
			cfg.Auth.RefreshTokenTTL = time.Hour
			cfg.Auth.Mode = config.AuthModeCookie
			cfg.Auth.CookieSameSite = tt.sameSite
			cfg.Auth.CookieSecure = tt.secure
			app, err := framework.New(
				framework.WithConfig(cfg),
				framework.WithDatabase(db),
				framework.WithLogger(zap.NewNop()),
			)
			if err != nil {
				t.Fatalf("framework.New() error = %v", err)
			}

			jar := fetchCSRF(t, app)
			registerCookieUser(t, app, jar, "attrs@example.com", testPassword)
			rec := loginCookieUser(t, app, jar, "attrs@example.com", testPassword)
			if rec.Code != http.StatusOK {
				t.Fatalf("login status = %d; body: %s", rec.Code, rec.Body.String())
			}

			cookies := map[string]*http.Cookie{}
			for _, c := range rec.Result().Cookies() {
				cookies[c.Name] = c
			}

			access, ok := cookies[auth.AccessCookieName]
			if !ok {
				t.Fatal("login response missing access cookie")
			}
			if !access.HttpOnly {
				t.Error("access cookie HttpOnly = false, want true")
			}
			if access.Secure != tt.secure {
				t.Errorf("access cookie Secure = %v, want %v", access.Secure, tt.secure)
			}
			if access.SameSite != tt.wantSameSite {
				t.Errorf("access cookie SameSite = %v, want %v", access.SameSite, tt.wantSameSite)
			}

			refresh, ok := cookies[auth.RefreshCookieName]
			if !ok {
				t.Fatal("login response missing refresh cookie")
			}
			if !refresh.HttpOnly {
				t.Error("refresh cookie HttpOnly = false, want true")
			}
			if refresh.Secure != tt.secure {
				t.Errorf("refresh cookie Secure = %v, want %v", refresh.Secure, tt.secure)
			}

			csrf := jar.cookies[auth.CSRFCookieName]
			if csrf == nil {
				t.Fatal("missing csrf cookie in jar")
			}
			if csrf.HttpOnly {
				t.Error("csrf cookie HttpOnly = true, want false (must be JS-readable)")
			}
		})
	}
}

func TestCSRFRejectsStateChangingRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieAuthApp(t)

	t.Run("no csrf cookie or header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"a@example.com","password":"correct-horse"}`))
		req.Header.Set("Content-Type", "application/json")
		app.Router().ServeHTTP(rec, req)
		assertCSRFRejected(t, rec)
	})

	t.Run("csrf cookie without header", func(t *testing.T) {
		jar := fetchCSRF(t, app)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"b@example.com","password":"correct-horse"}`))
		req.Header.Set("Content-Type", "application/json")
		jar.attach(req)
		app.Router().ServeHTTP(rec, req)
		assertCSRFRejected(t, rec)
	})

	t.Run("header does not match cookie", func(t *testing.T) {
		jar := fetchCSRF(t, app)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"c@example.com","password":"correct-horse"}`))
		req.Header.Set("Content-Type", "application/json")
		jar.attach(req)
		req.Header.Set(auth.CSRFHeaderName, "wrong-value")
		app.Router().ServeHTTP(rec, req)
		assertCSRFRejected(t, rec)
	})

	t.Run("forged unsigned token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"d@example.com","password":"correct-horse"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: "forged-token-not-signed"}) //nolint:gosec // G124: forged CSRF cookie for negative test.
		req.Header.Set(auth.CSRFHeaderName, "forged-token-not-signed")
		app.Router().ServeHTTP(rec, req)
		assertCSRFRejected(t, rec)
	})

	t.Run("valid double-submit token succeeds", func(t *testing.T) {
		jar := fetchCSRF(t, app)
		rec := doRequest(app, jar, http.MethodPost, "/api/v1/auth/register", `{"email":"e@example.com","password":"correct-horse"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("register with valid csrf status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestCSRFSafeMethodsAreExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieAuthApp(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	app.Router().ServeHTTP(rec, req)
	// No CSRF cookie yet, but GET is a safe method: CSRF middleware must not
	// block it (the 401 below comes from the missing session cookie, not a
	// CSRF rejection).
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /me status = %d, want 401 (missing session, not CSRF)", rec.Code)
	}
	if rec.Result().Header.Get("Set-Cookie") == "" {
		t.Fatal("safe GET request did not bootstrap a csrf cookie")
	}
	if got := rec.Result().Header.Get("WWW-Authenticate"); got != "" {
		t.Fatalf("cookie-mode 401 WWW-Authenticate = %q, want omit (not Bearer)", got)
	}
}

func TestCookieMe401OmitsBearerChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieAuthApp(t)

	t.Run("missing session cookie", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		app.Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
		}
		if got := rec.Result().Header.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want omit", got)
		}
		if !strings.Contains(rec.Body.String(), "missing session cookie") {
			t.Fatalf("body = %s, want missing session cookie", rec.Body.String())
		}
	})

	t.Run("invalid session cookie", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.AddCookie(&http.Cookie{Name: auth.AccessCookieName, Value: "not-a-jwt"}) //nolint:gosec // G124: request Cookie header only carries name/value.
		app.Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
		}
		if got := rec.Result().Header.Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want omit", got)
		}
		if !strings.Contains(rec.Body.String(), "invalid session cookie") {
			t.Fatalf("body = %s, want invalid session cookie", rec.Body.String())
		}
	})
}

func assertCSRFRejected(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
	var env contract.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, rec.Body.String())
	}
	if env.Body.Code != "authorization" {
		t.Fatalf("error.code = %q, want %q; body: %s", env.Body.Code, "authorization", rec.Body.String())
	}
}
