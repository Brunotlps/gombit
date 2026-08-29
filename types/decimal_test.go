package types_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gombit-dev/gombit/types"
)

func TestDecimalJSONRoundTripsAsString(t *testing.T) {
	d, err := types.NewDecimalFromString("19.99")
	if err != nil {
		t.Fatalf("NewDecimalFromString: %v", err)
	}
	b, err := json.Marshal(struct {
		Price types.Decimal `json:"price"`
	}{Price: d})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(b), `{"price":"19.99"}`; got != want {
		t.Fatalf("json = %s, want %s (a quoted string, no float rounding)", got, want)
	}

	var out struct {
		Price types.Decimal `json:"price"`
	}
	if err := json.Unmarshal([]byte(`{"price":"19.99"}`), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Price.Equal(d.Decimal) {
		t.Fatalf("round-trip = %s, want 19.99", out.Price.String())
	}
}

func TestDecimalHumaSchemaIsString(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := reg.Schema(reflect.TypeOf(types.Decimal{}), false, "Decimal")
	if s.Type != huma.TypeString {
		t.Fatalf("schema type = %q, want string (so OpenAPI/TS see a string)", s.Type)
	}
	if s.Format != "decimal" {
		t.Fatalf("schema format = %q, want decimal", s.Format)
	}
}

func TestDecimalGORMRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	type product struct {
		ID    uint
		Price types.Decimal `gorm:"type:decimal(19,4);not null"`
	}
	if err := db.AutoMigrate(&product{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	want, _ := types.NewDecimalFromString("1234.5600")
	row := product{Price: want}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	var got product
	if err := db.First(&got, row.ID).Error; err != nil {
		t.Fatalf("first: %v", err)
	}
	if !got.Price.Equal(want.Decimal) {
		t.Fatalf("stored price = %s, want %s", got.Price.String(), want.String())
	}
}
