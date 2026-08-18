package admin_test

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/admin"
	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/gin-gonic/gin"
)

type rowEnvelope struct {
	Data map[string]any `json:"data"`
}

type listEnvelope struct {
	Data []map[string]any   `json:"data"`
	Meta *contract.PageMeta `json:"meta"`
}

func TestResourceCRUDAndAuthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)

	create := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", `{"name":"Alpha","sku":"a-1","price":10}`)
	if create.Code != http.StatusOK {
		t.Fatalf("create status = %d; body: %s", create.Code, create.Body.String())
	}
	var created rowEnvelope
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data["name"] != "Alpha" {
		t.Fatalf("created name = %#v", created.Data["name"])
	}
	id := fmt.Sprint(asInt(created.Data["id"]))

	got := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets/"+id, "")
	if got.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body: %s", got.Code, got.Body.String())
	}

	patch := doRequest(app, jar, http.MethodPatch, "/api/v1/admin/resources/widgets/"+id, `{"note":"updated"}`)
	if patch.Code != http.StatusOK {
		t.Fatalf("patch status = %d; body: %s", patch.Code, patch.Body.String())
	}
	var updated rowEnvelope
	if err := json.Unmarshal(patch.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if updated.Data["note"] != "updated" {
		t.Fatalf("note = %#v", updated.Data["note"])
	}

	list := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d; body: %s", list.Code, list.Body.String())
	}
	var listed listEnvelope
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listed.Meta == nil || listed.Meta.Total != 1 {
		t.Fatalf("list meta = %+v", listed.Meta)
	}

	del := doRequest(app, jar, http.MethodDelete, "/api/v1/admin/resources/widgets/"+id, "")
	if del.Code != http.StatusOK {
		t.Fatalf("delete status = %d; body: %s", del.Code, del.Body.String())
	}
	missing := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets/"+id, "")
	assertError(t, missing, http.StatusNotFound, "not_found")
}

func TestResourceAnonymousUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	rec := doRequest(app, nil, http.MethodGet, "/api/v1/admin/resources/widgets", "")
	assertError(t, rec, http.StatusUnauthorized, "authentication")
}

func TestResourceNonSuperuserForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginUser(t, app, "staff@example.com", testPassword)
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets", "")
	assertError(t, rec, http.StatusForbidden, "authorization")
}

func TestResourceUnknownSlugNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/missing", "")
	assertError(t, rec, http.StatusNotFound, "not_found")
}

func TestResourceUnknownIDNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)
	rec := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets/9999", "")
	assertError(t, rec, http.StatusNotFound, "not_found")
}

func TestResourceDisabledActionForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app, func(o *admin.Options) {
		o.Actions.Delete = false
	})
	jar := loginSuperuser(t, app)
	create := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", `{"name":"Keep"}`)
	if create.Code != http.StatusOK {
		t.Fatalf("create status = %d; body: %s", create.Code, create.Body.String())
	}
	var created rowEnvelope
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id := fmt.Sprint(asInt(created.Data["id"]))
	rec := doRequest(app, jar, http.MethodDelete, "/api/v1/admin/resources/widgets/"+id, "")
	assertError(t, rec, http.StatusForbidden, "authorization")
}

func TestResourceValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)

	missing := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", `{}`)
	assertError(t, missing, http.StatusUnprocessableEntity, contract.CodeValidationError)
	env := decodeError(t, missing)
	if len(env.Fields["name"]) == 0 {
		t.Fatalf("fields.name missing; %#v", env.Fields)
	}

	readonly := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", `{"id":9,"name":"X"}`)
	assertError(t, readonly, http.StatusUnprocessableEntity, contract.CodeValidationError)

	unknown := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", `{"name":"X","nope":1}`)
	assertError(t, unknown, http.StatusUnprocessableEntity, contract.CodeValidationError)
}

func TestResourceListPaginationSearchOrderFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newCookieApp(t)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)

	fixtures := []string{
		`{"name":"Alpha","sku":"s-a","price":30}`,
		`{"name":"Beta","sku":"s-b","price":10}`,
		`{"name":"Gamma","sku":"s-a","price":20}`,
	}
	for _, body := range fixtures {
		rec := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("create %s status = %d; body: %s", body, rec.Code, rec.Body.String())
		}
	}

	page := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?page=1&per_page=2", "")
	if page.Code != http.StatusOK {
		t.Fatalf("page status = %d; body: %s", page.Code, page.Body.String())
	}
	var paged listEnvelope
	if err := json.Unmarshal(page.Body.Bytes(), &paged); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if paged.Meta == nil || paged.Meta.Page != 1 || paged.Meta.PerPage != 2 || paged.Meta.Total != 3 {
		t.Fatalf("page meta = %+v", paged.Meta)
	}
	if len(paged.Data) != 2 {
		t.Fatalf("page data len = %d, want 2", len(paged.Data))
	}

	search := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?search=Beta", "")
	var found listEnvelope
	if err := json.Unmarshal(search.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if found.Meta == nil || found.Meta.Total != 1 || len(found.Data) != 1 || found.Data[0]["name"] != "Beta" {
		t.Fatalf("search = %+v", found)
	}

	ordered := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?ordering=-price", "")
	var ranked listEnvelope
	if err := json.Unmarshal(ordered.Body.Bytes(), &ranked); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if len(ranked.Data) != 3 {
		t.Fatalf("order len = %d", len(ranked.Data))
	}
	if ranked.Data[0]["name"] != "Alpha" {
		t.Fatalf("first ordered = %#v, want Alpha", ranked.Data[0]["name"])
	}

	createdOrder := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?ordering=-created_at", "")
	if createdOrder.Code != http.StatusOK {
		t.Fatalf("ordering created_at status = %d; body: %s", createdOrder.Code, createdOrder.Body.String())
	}

	filtered := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?sku=s-a", "")
	var matches listEnvelope
	if err := json.Unmarshal(filtered.Body.Bytes(), &matches); err != nil {
		t.Fatalf("decode filter: %v", err)
	}
	if matches.Meta == nil || matches.Meta.Total != 2 {
		t.Fatalf("filter meta = %+v data=%+v", matches.Meta, matches.Data)
	}

	badOrder := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?ordering=note", "")
	assertError(t, badOrder, http.StatusUnprocessableEntity, contract.CodeValidationError)
}

func TestJWTModeDoesNotMountAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newJWTApp(t)
	rec := doRequest(app, nil, http.MethodGet, "/api/v1/admin/meta", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("JWT admin meta status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
	spec := doRequest(app, nil, http.MethodGet, "/openapi.json", "")
	if spec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d", spec.Code)
	}
	if strings.Contains(spec.Body.String(), "/api/v1/admin/meta") {
		t.Fatal("JWT OpenAPI includes admin routes")
	}
}

func TestHandlersDoNotImportReflect(t *testing.T) {
	t.Parallel()
	// Request handlers must not walk arbitrary Go types. Registration-time
	// reflect lives in fields.go (and register.go constructors).
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	forbidden := []string{
		"mount.go",
		"meta.go",
		"resources.go",
		"convert.go",
		"options.go",
		"errors.go",
		"registry.go",
	}
	fset := token.NewFileSet()
	for _, name := range forbidden {
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, spec := range file.Imports {
			if spec.Path != nil && spec.Path.Value == `"reflect"` {
				t.Errorf("%s imports reflect; request-time / handler files must not", name)
			}
		}
	}
}

func asInt(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case int:
		return int64(n)
	case int64:
		return n
	default:
		return 0
	}
}
