package contractspike

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pb33f/libopenapi"
	openapivalidator "github.com/pb33f/libopenapi-validator"
)

func TestTypedWidgetRoutes(t *testing.T) {
	server := NewServer()

	list := httptest.NewRecorder()
	server.Router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/widgets", nil))

	if list.Code != http.StatusOK {
		t.Fatalf("GET /widgets status = %d, want %d; body: %s", list.Code, http.StatusOK, list.Body.String())
	}

	var listBody SuccessEnvelope[[]Widget]
	if err := json.Unmarshal(list.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode GET /widgets response: %v", err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].ID != "widget-1" {
		t.Fatalf("GET /widgets data = %#v, want seeded widget", listBody.Data)
	}

	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"name":"Second widget","color":"green"}`))
	request.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(create, request)

	if create.Code != http.StatusOK {
		t.Fatalf("POST /widgets status = %d, want %d; body: %s", create.Code, http.StatusOK, create.Body.String())
	}

	var createBody SuccessEnvelope[Widget]
	if err := json.Unmarshal(create.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode POST /widgets response: %v", err)
	}
	if createBody.Data.ID != "widget-2" || createBody.Data.Name != "Second widget" {
		t.Fatalf("POST /widgets data = %#v, want created widget", createBody.Data)
	}
}

func TestRawRouteWorksAndIsAbsentFromOpenAPI(t *testing.T) {
	server := NewServer()

	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/raw/ping", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("GET /raw/ping status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var rawBody struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &rawBody); err != nil {
		t.Fatalf("decode GET /raw/ping response: %v", err)
	}
	if rawBody.Data.Status != "ok" {
		t.Fatalf("GET /raw/ping status body = %q, want ok", rawBody.Data.Status)
	}

	spec := openAPIDocument(t, server)
	paths := specObject[map[string]any](t, spec, "paths")
	if _, ok := paths["/raw/ping"]; ok {
		t.Fatalf("raw Gin route /raw/ping unexpectedly appears in OpenAPI paths")
	}
}

func TestOpenAPI31IncludesTypedWidgetSchemas(t *testing.T) {
	server := NewServer()

	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !json.Valid(response.Body.Bytes()) {
		t.Fatalf("GET /openapi.json returned invalid JSON")
	}

	spec := decodeObject(t, response.Body.Bytes())
	openapi := specString(t, spec, "openapi")
	if !strings.HasPrefix(openapi, "3.1.") {
		t.Fatalf("openapi = %q, want OpenAPI 3.1.x", openapi)
	}
	validateOpenAPI31Document(t, response.Body.Bytes())

	paths := specObject[map[string]any](t, spec, "paths")
	widgets := specObject[map[string]any](t, paths, "/widgets")

	get := specObject[map[string]any](t, widgets, "get")
	if specString(t, get, "operationId") != "list-widgets" {
		t.Fatalf("GET /widgets operationId = %q, want list-widgets", specString(t, get, "operationId"))
	}
	if _, ok := get["responses"]; !ok {
		t.Fatalf("GET /widgets missing responses")
	}

	post := specObject[map[string]any](t, widgets, "post")
	if specString(t, post, "operationId") != "create-widget" {
		t.Fatalf("POST /widgets operationId = %q, want create-widget", specString(t, post, "operationId"))
	}
	if _, ok := post["requestBody"]; !ok {
		t.Fatalf("POST /widgets missing requestBody schema")
	}
	if _, ok := post["responses"]; !ok {
		t.Fatalf("POST /widgets missing response schema")
	}

	specJSON := response.Body.String()
	for _, field := range []string{`"data"`, `"id"`, `"name"`, `"color"`} {
		if !strings.Contains(specJSON, field) {
			t.Fatalf("OpenAPI document missing typed schema field %s", field)
		}
	}
}

func TestOpenAPIJSONMatchesServedDocument(t *testing.T) {
	server := NewServer()

	generated, err := OpenAPIJSON(server.API)
	if err != nil {
		t.Fatalf("generate OpenAPI JSON: %v", err)
	}

	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d, want %d", response.Code, http.StatusOK)
	}

	if !jsonEqual(generated, response.Body.Bytes()) {
		t.Fatalf("generated OpenAPI JSON differs from served /openapi.json document")
	}
}

func TestCommittedOpenAPIJSONMatchesGeneratedDocument(t *testing.T) {
	generated, err := OpenAPIJSON(NewServer().API)
	if err != nil {
		t.Fatalf("generate OpenAPI JSON: %v", err)
	}

	committed, err := os.ReadFile("../../openapi.json")
	if err != nil {
		t.Fatalf("read committed openapi.json: %v", err)
	}

	if !jsonEqual(generated, committed) {
		t.Fatalf("committed openapi.json differs from generated document")
	}
}

func BenchmarkHumaGinListWidgets(b *testing.B) {
	router := NewServer().Router
	request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

	b.ReportAllocs()
	for range b.N {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("GET /widgets status = %d, want %d", response.Code, http.StatusOK)
		}
	}
}

func BenchmarkPlainGinListWidgets(b *testing.B) {
	router := plainGinWidgetRouter()
	request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

	b.ReportAllocs()
	for range b.N {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("GET /widgets status = %d, want %d", response.Code, http.StatusOK)
		}
	}
}

func plainGinWidgetRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	body := SuccessEnvelope[[]Widget]{
		Data: []Widget{{ID: "widget-1", Name: "First widget", Color: "blue"}},
	}
	router.GET("/widgets", func(c *gin.Context) {
		c.JSON(http.StatusOK, body)
	})

	return router
}

func openAPIDocument(t *testing.T, server *Server) map[string]any {
	t.Helper()

	data, err := OpenAPIJSON(server.API)
	if err != nil {
		t.Fatalf("generate OpenAPI JSON: %v", err)
	}

	return decodeObject(t, data)
}

func decodeObject(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var out map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&out); err != nil {
		t.Fatalf("decode JSON object: %v", err)
	}

	return out
}

func specObject[T any](t *testing.T, doc map[string]any, key string) T {
	t.Helper()

	value, ok := doc[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}

	typed, ok := value.(T)
	if !ok {
		t.Fatalf("key %q has type %T, want requested object type", key, value)
	}

	return typed
}

func specString(t *testing.T, doc map[string]any, key string) string {
	t.Helper()

	value, ok := doc[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}

	typed, ok := value.(string)
	if !ok {
		t.Fatalf("key %q has type %T, want string", key, value)
	}

	return typed
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

	return reflect.DeepEqual(leftValue, rightValue)
}

func validateOpenAPI31Document(t *testing.T, data []byte) {
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
