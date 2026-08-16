package framework

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
)

func TestAPIValidationReturnsD10FieldErrors(t *testing.T) {
	app := newTestApp(t)

	type createBody struct {
		Name string `json:"name" minLength:"1" maxLength:"80"`
	}
	type createInput struct {
		Body createBody
	}
	type createOutput struct {
		Body contract.Data[createBody]
	}

	prefix := app.Config().API.Prefix
	huma.Register(app.API(), huma.Operation{
		OperationID: "create-widget",
		Method:      http.MethodPost,
		Path:        prefix + "/widgets",
		Summary:     "Create a widget",
	}, func(ctx context.Context, input *createInput) (*createOutput, error) {
		return &createOutput{Body: contract.Data[createBody]{Data: input.Body}}, nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, prefix+"/widgets", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(RequestIDHeader, "req-framework-1")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
	if got := rec.Header().Get(RequestIDHeader); got != "req-framework-1" {
		t.Fatalf("%s = %q, want req-framework-1", RequestIDHeader, got)
	}

	var body contract.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, rec.Body.String())
	}
	if body.Body.Code != contract.CodeValidationError {
		t.Fatalf("code = %q, want %q", body.Body.Code, contract.CodeValidationError)
	}
	if body.Body.RequestID != "req-framework-1" {
		t.Fatalf("request_id = %q, want req-framework-1", body.Body.RequestID)
	}
	if len(body.Body.Fields["name"]) == 0 {
		t.Fatalf("fields.name missing; fields=%#v body=%s", body.Body.Fields, rec.Body.String())
	}
}

func TestAPIOpenAPIUsesD10ErrorSchema(t *testing.T) {
	app := newTestApp(t)

	huma.Register(app.API(), huma.Operation{
		OperationID: "ping",
		Method:      http.MethodGet,
		Path:        "/ping",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})

	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d; body: %s", rec.Code, rec.Body.String())
	}

	spec := rec.Body.String()
	if strings.Contains(spec, `"ErrorModel"`) {
		t.Fatalf("OpenAPI still advertises ErrorModel Problem Details schema")
	}
	if strings.Contains(spec, "application/problem+json") {
		t.Fatalf("OpenAPI still advertises application/problem+json")
	}
	if !strings.Contains(spec, `"ErrorEnvelope"`) {
		t.Fatalf("OpenAPI missing ErrorEnvelope schema; body: %s", spec)
	}
}

func TestAppExposesAPI(t *testing.T) {
	app := newTestApp(t)
	if app.API() == nil {
		t.Fatal("API() = nil, want huma.API")
	}
}
