package admin_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/admin"
	"github.com/gin-gonic/gin"
)

func TestRegisterMissingSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	err := admin.Register(app, Widget{}, admin.Options{})
	if err == nil || !strings.Contains(err.Error(), "missing slug") {
		t.Fatalf("Register() error = %v, want missing slug", err)
	}
}

func TestRegisterInvalidSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	err := admin.Register(app, Widget{}, admin.Options{Slug: "Widgets"})
	if err == nil || !strings.Contains(err.Error(), "invalid slug") {
		t.Fatalf("Register() error = %v, want invalid slug", err)
	}
}

func TestRegisterDuplicateSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	if err := admin.Register(app, Widget{}, widgetOptions()); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := admin.Register(app, Widget{}, widgetOptions())
	if err == nil || !strings.Contains(err.Error(), "duplicate slug") {
		t.Fatalf("second Register() error = %v, want duplicate slug", err)
	}
}

func TestRegisterRejectsJWTMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newJWTApp(t)
	err := admin.Register(app, Widget{}, widgetOptions())
	if err == nil || !strings.Contains(err.Error(), "cookie") {
		t.Fatalf("Register() on JWT app error = %v, want cookie-auth error", err)
	}
}

func TestRegisterDerivesPKAndEmptyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	if err := admin.Register(app, Widget{}, admin.Options{
		Slug:     "widgets",
		List:     []string{"name", "created_at"},
		Ordering: []string{"created_at"},
	}); err != nil {
		t.Fatalf("Register empty Fields: %v", err)
	}
}

func TestRegisterUnknownListField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	err := admin.Register(app, Widget{}, widgetOptions(func(o *admin.Options) {
		o.List = []string{"missing"}
	}))
	if err == nil || !strings.Contains(err.Error(), "list") {
		t.Fatalf("Register() error = %v, want unknown list field", err)
	}
}

func TestRegisterImplicitTimestampMissingOnModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	type Bare struct {
		ID   uint   `gorm:"primaryKey" json:"id"`
		Name string `json:"name"`
	}
	app := newCookieApp(t)
	if err := app.DB().AutoMigrate(&Bare{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	err := admin.Register(app, Bare{}, admin.Options{
		Slug:     "bares",
		Fields:   []admin.Field{{Name: "id", Type: admin.TypeInteger, ReadOnly: true}, {Name: "name", Type: admin.TypeString}},
		Ordering: []string{"created_at"},
	})
	if err == nil || !strings.Contains(err.Error(), "created_at") {
		t.Fatalf("Register() error = %v, want missing timestamp", err)
	}
}

func TestRegisterPKOverrideMustBeInFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	err := admin.Register(app, Widget{}, widgetOptions(func(o *admin.Options) {
		o.PK = "missing"
	}))
	if err == nil || !strings.Contains(err.Error(), "pk") {
		t.Fatalf("Register() error = %v, want pk not in Fields", err)
	}
}

func TestRegisterRejectsUnknownFieldType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	err := admin.Register(app, Widget{}, widgetOptions(func(o *admin.Options) {
		o.Fields = append(o.Fields, admin.Field{Name: "name", Type: "nope"})
	}))
	if err == nil {
		t.Fatal("Register() error = nil, want unknown type or duplicate")
	}
}

func TestFieldsFromUsesJSONNames(t *testing.T) {
	fields, err := admin.FieldsFrom(Widget{})
	if err != nil {
		t.Fatalf("FieldsFrom: %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("FieldsFrom returned no fields")
	}
	byName := map[string]admin.Field{}
	for _, f := range fields {
		byName[f.Name] = f
	}
	id, ok := byName["id"]
	if !ok {
		t.Fatalf("FieldsFrom missing id; fields=%v", fields)
	}
	if !id.ReadOnly {
		t.Fatal("id should be readonly")
	}
	if _, ok := byName["name"]; !ok {
		t.Fatalf("FieldsFrom missing name; fields=%v", fields)
	}
	if _, ok := byName["created_at"]; !ok {
		t.Fatalf("FieldsFrom missing created_at; fields=%v", fields)
	}
}

func TestFieldsFromIsRegistrationTimeOnly(t *testing.T) {
	// Compile-time documentation: FieldsFrom is exported for Register
	// callers, not handlers. The handler-import test locks the other half.
	_, err := admin.FieldsFrom(Widget{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterColumnMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	type Mapped struct {
		ID    uint   `gorm:"primaryKey" json:"id"`
		Title string `gorm:"column:title_text" json:"title"`
	}
	app := newCookieApp(t)
	if err := app.DB().AutoMigrate(&Mapped{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	err := admin.Register(app, Mapped{}, admin.Options{
		Slug: "mapped",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "title", Type: admin.TypeString, Column: "title_text", Required: true},
		},
		List: []string{"title"},
	})
	if err != nil {
		t.Fatalf("Register column mapping: %v", err)
	}
}

func TestRegisterErrorIsNotWrappedAsHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	err := admin.Register(app, Widget{}, admin.Options{})
	if err == nil {
		t.Fatal("expected error")
	}
	var env interface{ GetStatus() int }
	if errors.As(err, &env) {
		t.Fatalf("Register should not return an HTTP error, got %#v", err)
	}
}
