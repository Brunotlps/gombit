package admin_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"time"

	"github.com/LAA-Software-Engineering/gombit/admin"
	"github.com/LAA-Software-Engineering/gombit/auth"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/framework"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type catalogEnvelope struct {
	Data struct {
		Models []admin.ModelMeta `json:"models"`
	} `json:"data"`
	Meta *admin.CatalogAux `json:"meta"`
}

type modelEnvelope struct {
	Data admin.ModelMeta `json:"data"`
}

func TestMetaEmptyCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	jar := loginSuperuser(t, app)

	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/meta", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var env catalogEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v; body: %s", err, rec.Body.String())
	}
	if env.Data.Models == nil {
		t.Fatal("data.models is nil, want empty array")
	}
	if len(env.Data.Models) != 0 {
		t.Fatalf("data.models len = %d, want 0", len(env.Data.Models))
	}
	if env.Meta == nil || env.Meta.Auth == nil {
		t.Fatal("meta.auth is nil")
	}
	if env.Meta.Auth.Mode != "cookie" || env.Meta.Auth.Bootstrap != "is_superuser" {
		t.Fatalf("meta.auth = %+v", env.Meta.Auth)
	}
}

func TestMetaListsRegisteredModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)

	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/meta", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var env catalogEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v; body: %s", err, rec.Body.String())
	}
	if len(env.Data.Models) != 1 {
		t.Fatalf("data.models len = %d, want 1; body: %s", len(env.Data.Models), rec.Body.String())
	}
	got := env.Data.Models[0]
	if got.Slug != "widgets" || got.PK != "id" {
		t.Fatalf("model = %+v, want slug=widgets pk=id", got)
	}
	if len(got.Fields) == 0 {
		t.Fatal("fields is empty")
	}
	if got.Permissions.View != "admin.widgets.view" {
		t.Fatalf("permissions.view = %q", got.Permissions.View)
	}
	if !got.Actions.List || !got.Actions.Create {
		t.Fatalf("actions = %+v", got.Actions)
	}

	one := doRequest(app, jar, http.MethodGet, "/api/v1/admin/meta/widgets", "")
	if one.Code != http.StatusOK {
		t.Fatalf("GET meta/widgets status = %d; body: %s", one.Code, one.Body.String())
	}
	var single modelEnvelope
	if err := json.Unmarshal(one.Body.Bytes(), &single); err != nil {
		t.Fatalf("decode one: %v", err)
	}
	if single.Data.Slug != "widgets" {
		t.Fatalf("slug = %q", single.Data.Slug)
	}
}

func TestMetaRelationHasManyIsReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	type Category struct {
		ID   uint   `gorm:"primaryKey" json:"id"`
		Name string `json:"name"`
	}
	app := newCookieApp(t)
	if err := app.DB().AutoMigrate(&Category{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := admin.Register(app, Category{}, admin.Options{
		Slug: "categories",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "name", Type: admin.TypeString, Required: true},
			{
				Name: "widgets",
				Type: admin.TypeRelation,
				Related: &admin.Relation{
					Slug:       "widgets",
					Kind:       admin.RelHasMany,
					LabelField: "name",
				},
			},
		},
		List: []string{"name"},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	jar := loginSuperuser(t, app)
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/meta/categories", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var env modelEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var rel *admin.FieldMeta
	for i := range env.Data.Fields {
		if env.Data.Fields[i].Name == "widgets" {
			rel = &env.Data.Fields[i]
			break
		}
	}
	if rel == nil || rel.Related == nil || rel.Related.Kind != admin.RelHasMany {
		t.Fatalf("widgets relation = %+v", rel)
	}
	if !rel.ReadOnly {
		t.Fatal("has_many field should be readonly in meta")
	}
}

func TestMetaUnknownSlugNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/meta/missing", "")
	assertError(t, rec, http.StatusNotFound, "not_found")
}

func TestMetaAnonymousUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	rec := doRequest(app, nil, http.MethodGet, "/api/v1/admin/meta", "")
	assertError(t, rec, http.StatusUnauthorized, "authentication")
}

func TestMetaNonSuperuserForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginUser(t, app, "user@example.com", testPassword)
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/meta", "")
	assertError(t, rec, http.StatusForbidden, "authorization")
}

func TestMetaHonorsAPIPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openSQLite(t)
	if err := auth.Migrate(db.DB); err != nil {
		t.Fatalf("auth.Migrate: %v", err)
	}
	if err := db.AutoMigrate(&Widget{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.Auth.JWTSecret = testJWTSecret
	cfg.Auth.BcryptCost = bcrypt.MinCost
	cfg.Auth.AccessTokenTTL = time.Minute
	cfg.Auth.RefreshTokenTTL = time.Hour
	cfg.Auth.Mode = config.AuthModeCookie
	cfg.API.Prefix = "/svc/v2"
	app, err := framework.New(
		framework.WithConfig(cfg),
		framework.WithDatabase(db),
		framework.WithLogger(zap.NewNop()),
	)
	if err != nil {
		t.Fatalf("framework.New: %v", err)
	}
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)
	rec := doRequest(app, jar, http.MethodGet, "/svc/v2/admin/meta", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminRoutesAppearInOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	rec := doRequest(app, nil, http.MethodGet, "/openapi.json", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, path := range []string{
		"/api/v1/admin/meta",
		"/api/v1/admin/meta/{slug}",
		"/api/v1/admin/resources/{slug}",
		"/api/v1/admin/resources/{slug}/{id}",
	} {
		if !strings.Contains(body, path) {
			t.Fatalf("openapi.json missing %s", path)
		}
	}
}
