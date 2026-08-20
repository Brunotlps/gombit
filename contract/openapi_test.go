package contract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/pb33f/libopenapi"
	openapivalidator "github.com/pb33f/libopenapi-validator"
)

func TestOpenAPIJSONMatchesServedDocument(t *testing.T) {
	router, api := newContractAPI(t)

	huma.Register(api, huma.Operation{
		OperationID: "list-widgets",
		Method:      http.MethodGet,
		Path:        "/widgets",
		Summary:     "List widgets",
	}, func(ctx context.Context, input *struct{}) (*struct {
		Body contract.Data[[]string]
	}, error) {
		return &struct {
			Body contract.Data[[]string]
		}{Body: contract.Data[[]string]{Data: []string{"widget-1"}}}, nil
	})

	generated, err := contract.OpenAPIJSON(api)
	if err != nil {
		t.Fatalf("OpenAPIJSON() error = %v", err)
	}
	validateOpenAPI31(t, generated)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if !jsonEqual(generated, rec.Body.Bytes()) {
		t.Fatalf("generated OpenAPI JSON differs from served /openapi.json")
	}

	var spec map[string]any
	if err := json.Unmarshal(generated, &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	openapi, _ := spec["openapi"].(string)
	if !strings.HasPrefix(openapi, "3.1.") {
		t.Fatalf("openapi = %q, want OpenAPI 3.1.x", openapi)
	}
	paths, _ := spec["paths"].(map[string]any)
	if _, ok := paths["/widgets"]; !ok {
		t.Fatalf("spec missing /widgets; paths=%v", paths)
	}
}

func TestWriteOpenAPIWritesDocument(t *testing.T) {
	_, api := newContractAPI(t)
	huma.Register(api, huma.Operation{
		OperationID: "ping",
		Method:      http.MethodGet,
		Path:        "/ping",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})

	path := filepath.Join(t.TempDir(), "out", "openapi.json")
	if err := contract.WriteOpenAPI(path, api); err != nil {
		t.Fatalf("WriteOpenAPI() error = %v", err)
	}
	// #nosec G304 -- path is built from t.TempDir
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written spec: %v", err)
	}
	want, err := contract.OpenAPIJSON(api)
	if err != nil {
		t.Fatalf("OpenAPIJSON() error = %v", err)
	}
	if !jsonEqual(got, append(want, '\n')) && !jsonEqual(got, want) {
		t.Fatalf("written spec does not match OpenAPIJSON()")
	}
	if !strings.HasSuffix(string(got), "\n") {
		t.Fatal("written spec missing trailing newline")
	}
}

func TestOpenAPIJSONRequiresAPI(t *testing.T) {
	if _, err := contract.OpenAPIJSON(nil); err == nil {
		t.Fatal("OpenAPIJSON(nil) error = nil, want error")
	}
	if err := contract.WriteOpenAPIFile("", []byte(`{}`)); err == nil {
		t.Fatal("WriteOpenAPIFile empty path error = nil, want error")
	}
	if err := contract.WriteOpenAPIFile("openapi.json", nil); err == nil {
		t.Fatal("WriteOpenAPIFile empty spec error = nil, want error")
	}
}

func TestDocsServesSwaggerUI(t *testing.T) {
	router, api := newContractAPI(t)
	huma.Register(api, huma.Operation{
		OperationID: "list-widgets",
		Method:      http.MethodGet,
		Path:        "/widgets",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /docs status = %d; body: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("GET /docs Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Fatalf("GET /docs missing swagger-ui; body: %s", body)
	}
	if !strings.Contains(body, "/openapi") {
		t.Fatalf("GET /docs missing OpenAPI URL for try-it-out; body: %s", body)
	}
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "unpkg.com/swagger-ui-dist") {
		t.Fatalf("GET /docs CSP = %q, want Swagger UI script source so try-it-out can load", csp)
	}
	if !strings.Contains(csp, "connect-src") || !strings.Contains(csp, "'self'") {
		t.Fatalf("GET /docs CSP = %q, want connect-src 'self' for try-it-out", csp)
	}
	if !strings.Contains(body, "Try it out") && !strings.Contains(strings.ToLower(body), "swagger") {
		t.Fatalf("GET /docs missing try-it-out UI markers; body: %s", body)
	}
}

func TestRawGinRouteAbsentFromOpenAPI(t *testing.T) {
	router, api := newContractAPI(t)
	huma.Register(api, huma.Operation{
		OperationID: "list-widgets",
		Method:      http.MethodGet,
		Path:        "/widgets",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})
	router.GET("/raw/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/raw/ping", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /raw/ping status = %d, want %d", rec.Code, http.StatusOK)
	}

	spec, err := contract.OpenAPIJSON(api)
	if err != nil {
		t.Fatalf("OpenAPIJSON() error = %v", err)
	}
	if strings.Contains(string(spec), "/raw/ping") {
		t.Fatal("raw Gin route /raw/ping unexpectedly appears in OpenAPI")
	}
	if strings.Contains(string(spec), "\"/docs\"") {
		t.Fatal("docs UI unexpectedly appears in OpenAPI paths")
	}
}

func newContractAPI(t *testing.T) (*gin.Engine, huma.API) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, contract.HumaConfig("contract-test", "0.0.0"))
	return router, api
}

func validateOpenAPI31(t *testing.T, data []byte) {
	t.Helper()

	document, err := libopenapi.NewDocument(data)
	if err != nil {
		t.Fatalf("parse OpenAPI document: %v", err)
	}
	validator, validatorErrs := openapivalidator.NewValidator(document)
	if len(validatorErrs) > 0 {
		t.Fatalf("create OpenAPI validator: %v", validatorErrs)
	}
	valid, documentErrs := validator.ValidateDocument()
	if !valid || len(documentErrs) > 0 {
		t.Fatalf("validate OpenAPI 3.1 document: valid=%v errors=%v", valid, documentErrs)
	}
}

func jsonEqual(left, right []byte) bool {
	var leftValue any
	var rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}
	return jsonEqualValue(leftValue, rightValue)
}

func jsonEqualValue(left, right any) bool {
	leftBytes, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightBytes, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return string(leftBytes) == string(rightBytes)
}
