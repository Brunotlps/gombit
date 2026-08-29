package admin_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gombit-dev/gombit/admin"
	"github.com/gombit-dev/gombit/framework"
)

type relWarehouse struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
}

func (relWarehouse) TableName() string { return "warehouses" }

type relEngine struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Name       string         `json:"name"`
	Warehouses []relWarehouse `gorm:"many2many:engine_warehouses;" json:"warehouses"`
}

func (relEngine) TableName() string { return "engines" }

// idsOf pulls a []int64 out of a JSON relation field for order-independent
// comparison.
func idsOf(t *testing.T, v any) []int64 {
	t.Helper()
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("field = %#v, want an array of ids", v)
	}
	out := make([]int64, 0, len(arr))
	for _, e := range arr {
		out = append(out, asInt(e))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func TestResourceManyToMany(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	if err := app.DB().AutoMigrate(&relWarehouse{}, &relEngine{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := admin.Register(app, relWarehouse{}, admin.Options{
		Slug: "warehouses",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "name", Type: admin.TypeString, Required: true},
		},
	}); err != nil {
		t.Fatalf("Register warehouse: %v", err)
	}
	if err := admin.Register(app, relEngine{}, admin.Options{
		Slug: "engines",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "name", Type: admin.TypeString, Required: true},
			{Name: "warehouses", Type: admin.TypeRelation, Related: &admin.Relation{
				Kind: admin.RelManyToMany, Slug: "warehouses", LabelField: "name",
			}},
		},
	}); err != nil {
		t.Fatalf("Register engine: %v", err)
	}
	jar := loginSuperuser(t, app)

	w1 := createWarehouse(t, app, jar, "North")
	w2 := createWarehouse(t, app, jar, "South")

	// Create an engine with two warehouses.
	create := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/engines",
		fmt.Sprintf(`{"name":"V8","warehouses":[%d,%d]}`, w1, w2))
	if create.Code != http.StatusOK {
		t.Fatalf("create engine status = %d; body: %s", create.Code, create.Body.String())
	}
	var created rowEnvelope
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if got := idsOf(t, created.Data["warehouses"]); len(got) != 2 || got[0] != w1 || got[1] != w2 {
		t.Fatalf("created warehouses = %v, want [%d %d]", got, w1, w2)
	}
	engineID := asInt(created.Data["id"])
	path := fmt.Sprintf("/api/v1/admin/resources/engines/%d", engineID)

	// Read it back — the join table round-trips.
	get := doRequest(app, jar, http.MethodGet, path, "")
	var got rowEnvelope
	if err := json.Unmarshal(get.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if ids := idsOf(t, got.Data["warehouses"]); len(ids) != 2 {
		t.Fatalf("get warehouses = %v, want 2", ids)
	}

	// Shrink to one warehouse.
	patch := doRequest(app, jar, http.MethodPatch, path, fmt.Sprintf(`{"warehouses":[%d]}`, w1))
	if patch.Code != http.StatusOK {
		t.Fatalf("patch status = %d; body: %s", patch.Code, patch.Body.String())
	}
	var patched rowEnvelope
	_ = json.Unmarshal(patch.Body.Bytes(), &patched)
	if ids := idsOf(t, patched.Data["warehouses"]); len(ids) != 1 || ids[0] != w1 {
		t.Fatalf("patched warehouses = %v, want [%d]", ids, w1)
	}

	// A non-existent related id is a 422, not a silent no-op.
	bad := doRequest(app, jar, http.MethodPatch, path, `{"warehouses":[999999]}`)
	assertError(t, bad, http.StatusUnprocessableEntity, "validation_error")

	// Omitting the relation on PATCH leaves it unchanged (partial update).
	rename := doRequest(app, jar, http.MethodPatch, path, `{"name":"V8-b"}`)
	if rename.Code != http.StatusOK {
		t.Fatalf("rename status = %d; body: %s", rename.Code, rename.Body.String())
	}
	var renamed rowEnvelope
	_ = json.Unmarshal(rename.Body.Bytes(), &renamed)
	if ids := idsOf(t, renamed.Data["warehouses"]); len(ids) != 1 || ids[0] != w1 {
		t.Fatalf("after rename warehouses = %v, want unchanged [%d]", ids, w1)
	}

	// Clearing to an empty list removes all join rows.
	clear := doRequest(app, jar, http.MethodPatch, path, `{"warehouses":[]}`)
	if clear.Code != http.StatusOK {
		t.Fatalf("clear status = %d; body: %s", clear.Code, clear.Body.String())
	}
	var cleared rowEnvelope
	_ = json.Unmarshal(clear.Body.Bytes(), &cleared)
	if ids := idsOf(t, cleared.Data["warehouses"]); len(ids) != 0 {
		t.Fatalf("cleared warehouses = %v, want empty", ids)
	}
}

func createWarehouse(t *testing.T, app *framework.App, jar *cookieJar, name string) int64 {
	t.Helper()
	rec := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/warehouses",
		fmt.Sprintf(`{"name":%q}`, name))
	if rec.Code != http.StatusOK {
		t.Fatalf("create warehouse status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var env rowEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode warehouse: %v", err)
	}
	return asInt(env.Data["id"])
}

// TestManyToManyAutoDerivation verifies FieldsFrom emits a relation field for a
// many-to-many association instead of dropping it (#221/#223).
func TestManyToManyAutoDerivation(t *testing.T) {
	fields, err := admin.FieldsFrom(relEngine{})
	if err != nil {
		t.Fatalf("FieldsFrom: %v", err)
	}
	var found *admin.Field
	for i := range fields {
		if fields[i].Name == "warehouses" {
			found = &fields[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("derived fields %v missing warehouses relation", fields)
	}
	if found.Type != admin.TypeRelation || found.Related == nil || found.Related.Kind != admin.RelManyToMany {
		t.Fatalf("warehouses field = %+v, want many_to_many relation", found)
	}
	if found.Related.Slug != "warehouses" {
		t.Fatalf("warehouses slug = %q, want warehouses (related table)", found.Related.Slug)
	}
}
