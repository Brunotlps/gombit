package contract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

func TestValidationReturnsD10FieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, contract.HumaConfig("contract-test", "0.0.0"))

	type createBody struct {
		Name string `json:"name" minLength:"1" maxLength:"80"`
	}
	type createInput struct {
		Body createBody
	}
	type createOutput struct {
		Body contract.Data[createBody]
	}

	huma.Register(api, huma.Operation{
		OperationID: "create-widget",
		Method:      http.MethodPost,
		Path:        "/widgets",
		Summary:     "Create a widget",
	}, func(ctx context.Context, input *createInput) (*createOutput, error) {
		return &createOutput{Body: contract.Data[createBody]{Data: input.Body}}, nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if strings.Contains(ct, "problem+json") {
		t.Fatalf("Content-Type = %q, must not be problem+json", ct)
	}

	var body contract.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, rec.Body.String())
	}
	if body.Body.Code != contract.CodeValidationError {
		t.Fatalf("code = %q, want %q", body.Body.Code, contract.CodeValidationError)
	}
	if body.Body.RequestID != "req-test-1" {
		t.Fatalf("request_id = %q, want req-test-1", body.Body.RequestID)
	}
	if len(body.Body.Fields["name"]) == 0 {
		t.Fatalf("fields.name missing; fields=%#v body=%s", body.Body.Fields, rec.Body.String())
	}
}

func TestHumaConfigDisablesDocsAndSchemas(t *testing.T) {
	cfg := contract.HumaConfig("Demo", "1.2.3")
	if cfg.DocsPath != "" {
		t.Fatalf("DocsPath = %q, want empty", cfg.DocsPath)
	}
	if cfg.SchemasPath != "" {
		t.Fatalf("SchemasPath = %q, want empty", cfg.SchemasPath)
	}
	if cfg.OpenAPIPath != "/openapi" {
		t.Fatalf("OpenAPIPath = %q, want /openapi", cfg.OpenAPIPath)
	}
	if cfg.Info == nil || cfg.Info.Title != "Demo" || cfg.Info.Version != "1.2.3" {
		t.Fatalf("Info = %#v, want Demo/1.2.3", cfg.Info)
	}
}

func TestOpenAPIUsesD10ErrorSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, contract.HumaConfig("contract-test", "0.0.0"))

	huma.Register(api, huma.Operation{
		OperationID: "ping",
		Method:      http.MethodGet,
		Path:        "/ping",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d; body: %s", rec.Code, rec.Body.String())
	}

	spec := rec.Body.String()
	if !strings.Contains(spec, `"ErrorEnvelope"`) && !strings.Contains(spec, `"error"`) {
		t.Fatalf("OpenAPI missing D10 error shape markers; body: %s", spec)
	}
	if strings.Contains(spec, `"ErrorModel"`) {
		t.Fatalf("OpenAPI still advertises ErrorModel Problem Details schema")
	}
	if strings.Contains(spec, "application/problem+json") {
		t.Fatalf("OpenAPI still advertises application/problem+json")
	}
}
