package admin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/admin"
	"github.com/gombit-dev/gombit/auth"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/contract"
	"github.com/gombit-dev/gombit/database"
	"github.com/gombit-dev/gombit/framework"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	testJWTSecret = "admin-test-jwt-secret-32-bytes-ok" // #nosec G101 -- fake test secret.
	testPassword  = "correct-horse"
)

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

type Widget struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	Price     int       `json:"price"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func widgetOptions(overrides ...func(*admin.Options)) admin.Options {
	opts := admin.Options{
		Slug:     "widgets",
		Singular: "Widget",
		Plural:   "Widgets",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "name", Type: admin.TypeString, Required: true},
			{Name: "sku", Type: admin.TypeString},
			{Name: "price", Type: admin.TypeInteger},
			{Name: "note", Type: admin.TypeText},
		},
		List:     []string{"name", "sku", "price"},
		Search:   []string{"name", "sku"},
		Filter:   []string{"sku"},
		Ordering: []string{"name", "price", "created_at"},
		Actions: admin.Actions{
			List: true, Detail: true, Create: true, Update: true, Delete: true,
		},
	}
	for _, fn := range overrides {
		fn(&opts)
	}
	return opts
}

func openSQLite(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:" + filepath.Join(t.TempDir(), "admin.db") + "?cache=shared&_fk=1",
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newCookieApp(t *testing.T) *framework.App {
	t.Helper()
	return newCookieAppWithDB(t, openSQLite(t))
}

func newCookieAppWithDB(t *testing.T, db *database.DB) *framework.App {
	t.Helper()
	if err := auth.Migrate(db.DB); err != nil {
		t.Fatalf("auth.Migrate() error = %v", err)
	}
	if err := db.AutoMigrate(&Widget{}); err != nil {
		t.Fatalf("AutoMigrate(Widget) error = %v", err)
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

func newJWTApp(t *testing.T) *framework.App {
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

func apiPrefix(app *framework.App) string {
	prefix := strings.TrimSuffix(app.Config().API.Prefix, "/")
	if prefix == "" {
		return "/api/v1"
	}
	return prefix
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
	rec := doRequest(app, jar, http.MethodGet, apiPrefix(app)+"/auth/csrf", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /auth/csrf status = %d; body: %s", rec.Code, rec.Body.String())
	}
	return jar
}

func loginSuperuser(t *testing.T, app *framework.App) *cookieJar {
	t.Helper()
	svc, err := auth.NewService(app.DB(), app.Config())
	if err != nil {
		t.Fatalf("auth.NewService: %v", err)
	}
	email := "admin-" + strings.ReplaceAll(t.Name(), "/", "-") + "@example.com"
	if _, err := svc.CreateSuperuser(context.Background(), email, testPassword); err != nil {
		t.Fatalf("CreateSuperuser: %v", err)
	}
	jar := fetchCSRF(t, app)
	body, err := json.Marshal(map[string]string{"email": email, "password": testPassword})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := doRequest(app, jar, http.MethodPost, apiPrefix(app)+"/auth/login", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d; body: %s", rec.Code, rec.Body.String())
	}
	return jar
}

func loginUser(t *testing.T, app *framework.App, email, password string) *cookieJar {
	t.Helper()
	jar := fetchCSRF(t, app)
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := doRequest(app, jar, http.MethodPost, apiPrefix(app)+"/auth/register", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("register status = %d; body: %s", rec.Code, rec.Body.String())
	}
	rec = doRequest(app, jar, http.MethodPost, apiPrefix(app)+"/auth/login", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d; body: %s", rec.Code, rec.Body.String())
	}
	return jar
}

func grantGroupPermission(t *testing.T, app *framework.App, email, key string) {
	t.Helper()
	ctx := context.Background()
	var user auth.User
	if err := app.DB().WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("load user %s: %v", email, err)
	}
	permission, err := auth.EnsurePermission(ctx, app.DB(), key, "")
	if err != nil {
		t.Fatalf("EnsurePermission(%s): %v", key, err)
	}
	group, err := auth.EnsureGroup(ctx, app.DB(), "group-"+strings.ReplaceAll(t.Name(), "/", "-"))
	if err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	if err := auth.GrantPermissionToGroup(ctx, app.DB(), &group, &permission); err != nil {
		t.Fatalf("GrantPermissionToGroup: %v", err)
	}
	if err := auth.AddUserToGroup(ctx, app.DB(), &user, &group); err != nil {
		t.Fatalf("AddUserToGroup: %v", err)
	}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) contract.ErrorBody {
	t.Helper()
	var env contract.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, rec.Body.String())
	}
	return env.Body
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, status, rec.Body.String())
	}
	got := decodeError(t, rec)
	if got.Code != code {
		t.Fatalf("error.code = %q, want %q; body: %s", got.Code, code, rec.Body.String())
	}
}

func registerWidgets(t *testing.T, app *framework.App, overrides ...func(*admin.Options)) {
	t.Helper()
	if err := admin.Register(app, Widget{}, widgetOptions(overrides...)); err != nil {
		t.Fatalf("admin.Register: %v", err)
	}
}
