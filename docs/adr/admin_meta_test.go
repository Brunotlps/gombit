package adr_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Types below document the ADMIN-0 introspection envelope sketched in
// ADR-013. They are test-only and are not a public admin API.

type metaEnvelope struct {
	Data metaData `json:"data"`
	Meta *metaAux `json:"meta"`
}

type metaData struct {
	Models []metaModel `json:"models"`
}

type metaAux struct {
	Auth *metaAuth `json:"auth"`
}

type metaAuth struct {
	Mode      string `json:"mode"`
	Bootstrap string `json:"bootstrap"`
}

type metaModel struct {
	Slug        string            `json:"slug"`
	Singular    string            `json:"singular"`
	Plural      string            `json:"plural"`
	PK          string            `json:"pk"`
	Fields      []metaField       `json:"fields"`
	List        []string          `json:"list"`
	Search      []string          `json:"search"`
	Filter      []string          `json:"filter"`
	Ordering    []string          `json:"ordering"`
	Actions     metaActions       `json:"actions"`
	Permissions map[string]string `json:"permissions"`
}

type metaField struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	Required bool         `json:"required"`
	ReadOnly bool         `json:"readonly"`
	Related  *metaRelated `json:"related"`
}

type metaRelated struct {
	Slug       string `json:"slug"`
	Kind       string `json:"kind"`
	LabelField string `json:"label_field"`
}

type metaActions struct {
	List   bool `json:"list"`
	Detail bool `json:"detail"`
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

func TestAdminMetaFixtureUnmarshal(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "admin-meta.json")
	raw, err := os.ReadFile(path) // #nosec G304 -- test fixture path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var env metaEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", path, err)
	}

	if len(env.Data.Models) != 2 {
		t.Fatalf("data.models len = %d, want 2", len(env.Data.Models))
	}
	if env.Meta == nil || env.Meta.Auth == nil {
		t.Fatal("meta.auth is nil")
	}
	if env.Meta.Auth.Mode != "cookie" {
		t.Errorf("meta.auth.mode = %q, want %q", env.Meta.Auth.Mode, "cookie")
	}
	if env.Meta.Auth.Bootstrap != "is_superuser" {
		t.Errorf("meta.auth.bootstrap = %q, want %q", env.Meta.Auth.Bootstrap, "is_superuser")
	}

	products := env.Data.Models[0]
	if products.Slug != "products" {
		t.Errorf("models[0].slug = %q, want %q", products.Slug, "products")
	}
	if products.PK != "id" {
		t.Errorf("models[0].pk = %q, want %q", products.PK, "id")
	}
	if !products.Actions.List || !products.Actions.Create {
		t.Errorf("models[0].actions = %+v, want list and create enabled", products.Actions)
	}
	if got := products.Permissions["view"]; got != "admin.products.view" {
		t.Errorf("models[0].permissions.view = %q, want %q", got, "admin.products.view")
	}

	var relation *metaField
	for i := range products.Fields {
		if products.Fields[i].Name == "category_id" {
			relation = &products.Fields[i]
			break
		}
	}
	if relation == nil {
		t.Fatal("models[0] missing category_id field")
	}
	if relation.Type != "relation" {
		t.Errorf("category_id.type = %q, want %q", relation.Type, "relation")
	}
	if relation.Related == nil {
		t.Fatal("category_id.related is nil")
	}
	if relation.Related.Slug != "categories" || relation.Related.Kind != "belongs_to" {
		t.Errorf("category_id.related = %+v, want slug=categories kind=belongs_to", relation.Related)
	}

	if env.Data.Models[1].Slug != "categories" {
		t.Errorf("models[1].slug = %q, want %q", env.Data.Models[1].Slug, "categories")
	}
}
