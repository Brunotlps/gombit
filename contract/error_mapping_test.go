package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

func TestHumaHandlerReturnsNotFoundEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, HumaConfig("contract-test", "0.0.0"))

	type getInput struct {
		ID string `path:"id"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "get-widget",
		Method:      http.MethodGet,
		Path:        "/widgets/{id}",
	}, func(ctx context.Context, input *getInput) (*struct {
		Body Data[string]
	}, error) {
		return nil, WithContext(ctx, NotFound("widget not found"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/widgets/missing", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}

	var body ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v; body: %s", err, rec.Body.String())
	}
	if body.Body.Code != "not_found" {
		t.Fatalf("code = %q, want not_found", body.Body.Code)
	}
	if body.Body.Message != "widget not found" {
		t.Fatalf("message = %q", body.Body.Message)
	}
	if body.Body.RequestID != "req-test-1" {
		t.Fatalf("request_id = %q, want req-test-1 (from TestMain Install)", body.Body.RequestID)
	}
}

func TestConflictWithFieldsKeepsCategoryCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, HumaConfig("contract-test", "0.0.0"))

	huma.Register(api, huma.Operation{
		OperationID: "create-dup",
		Method:      http.MethodPost,
		Path:        "/widgets",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, WithContext(ctx, Conflict("duplicate name").WithFields(map[string][]string{
			"name": {"already exists"},
		}))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/widgets", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
	var body ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v; body: %s", err, rec.Body.String())
	}
	if body.Body.Code != "conflict" {
		t.Fatalf("code = %q, want conflict", body.Body.Code)
	}
	if len(body.Body.Fields["name"]) == 0 {
		t.Fatalf("fields.name missing: %#v", body.Body.Fields)
	}
}

func TestDataMetaPageMetaAppearsInOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, HumaConfig("contract-test", "0.0.0"))

	type item struct {
		Name string `json:"name"`
	}
	type listOut struct {
		Body DataMeta[[]item, PageMeta]
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-items",
		Method:      http.MethodGet,
		Path:        "/items",
	}, func(ctx context.Context, input *struct{}) (*listOut, error) {
		return &listOut{Body: DataMeta[[]item, PageMeta]{
			Data: []item{{Name: "a"}},
			Meta: &PageMeta{Page: 1, PerPage: 20, Total: 1},
		}}, nil
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d; body: %s", rec.Code, rec.Body.String())
	}
	spec := rec.Body.String()
	for _, want := range []string{`"page"`, `"per_page"`, `"total"`, `"PageMeta"`} {
		if !strings.Contains(spec, want) {
			t.Fatalf("OpenAPI missing %s; body: %s", want, spec)
		}
	}
}
