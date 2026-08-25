package contractspike

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// BenchUser is the JSON resource shared by the path-parameter and POST
// framework-tax microbenchmarks (issue #141 "1. Go abstraction-overhead
// microbenchmarks"). Each stack serializes it idiomatically: net/http and Gin
// return it bare; Huma and Gombit wrap it in the D10 SuccessEnvelope, since
// that envelope is itself part of what those two stacks cost.
type BenchUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// CreateBenchUserBody is the validated POST /users request body shared by the
// Huma and Gombit stacks. The pattern tag is enforced by Huma's JSON-schema
// validator at request time, not just documented in the OpenAPI schema, so
// the "invalid POST" benchmark scenario exercises real validation work.
type CreateBenchUserBody struct {
	Name  string `json:"name" minLength:"1" maxLength:"80" example:"Ada Lovelace" doc:"Human-readable user name"`
	Email string `json:"email" pattern:"^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$" example:"ada@example.com" doc:"Contact email address"`
}

// ginCreateUserRequest is the equivalent idiomatic Gin request body, using
// Gin's own binding-tag validation (go-playground/validator under the hood)
// instead of Huma's JSON-schema validation.
type ginCreateUserRequest struct {
	Name  string `json:"name" binding:"required,max=80"`
	Email string `json:"email" binding:"required,email"`
}

type plaintextOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

type jsonMessageOutput struct {
	Body SuccessEnvelope[map[string]string]
}

type getBenchUserInput struct {
	ID string `path:"id"`
}

type getBenchUserOutput struct {
	Body SuccessEnvelope[BenchUser]
}

type createBenchUserInput struct {
	Body CreateBenchUserBody
}

type createBenchUserOutput struct {
	Body SuccessEnvelope[BenchUser]
}

var (
	errBenchUserNameRequired  = errors.New("name is required")
	errBenchUserEmailRequired = errors.New("email must be a valid address")
)

// NewNetHTTPHandler returns the plain net/http baseline for the
// framework-tax benchmark matrix: no router, no framework, hand-written JSON
// encode/decode and manual validation. This is the floor every other stack
// is measured against.
func NewNetHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /plaintext", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("GET /json", func(w http.ResponseWriter, r *http.Request) {
		writeBenchJSON(w, http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeBenchJSON(w, http.StatusOK, BenchUser{ID: r.PathValue("id"), Name: "Ada Lovelace"})
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var body CreateBenchUserBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeBenchJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if err := validateBenchUserBody(body); err != nil {
			writeBenchJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeBenchJSON(w, http.StatusCreated, BenchUser{ID: "user-1", Name: body.Name, Email: body.Email})
	})

	return mux
}

func validateBenchUserBody(body CreateBenchUserBody) error {
	if strings.TrimSpace(body.Name) == "" {
		return errBenchUserNameRequired
	}
	if strings.TrimSpace(body.Email) == "" || !strings.Contains(body.Email, "@") {
		return errBenchUserEmailRequired
	}
	return nil
}

func writeBenchJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// NewBenchGinRouter returns the idiomatic plain-Gin baseline: Gin routing and
// binding-tag validation, no Huma, no framework.
func NewBenchGinRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/plaintext", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	router.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	router.GET("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, BenchUser{ID: c.Param("id"), Name: "Ada Lovelace"})
	})

	router.POST("/users", func(c *gin.Context) {
		var body ginCreateUserRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, BenchUser{ID: "user-1", Name: body.Name, Email: body.Email})
	})

	return router
}

// NewBenchHumaGinServer returns a bare Huma-over-Gin router carrying the same
// five routes as NewNetHTTPHandler/NewBenchGinRouter. It is intentionally
// isolated from the widget routes registered by NewServer so the committed
// root openapi.json (generated from NewServer, see TestCommittedOpenAPIJSONMatchesGeneratedDocument)
// never drifts because of benchmark-only routes.
func NewBenchHumaGinServer() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, Config())
	RegisterBenchRoutes(api)
	return router
}

// RegisterBenchRoutes registers the five framework-tax benchmark scenarios
// (plaintext, JSON, path parameter, validated POST, invalid POST) onto api.
// It is exported so internal/contractspike/gombitbench can share the exact
// same handlers for the Gombit runtime stack: that package constructs a real
// framework.App, which calls contract.Install and permanently replaces
// Huma's process-global huma.NewError (see contract.Install docs and
// docs/spikes/M0-2_HUMA_GIN_SPIKE.md). Keeping the Gombit stack in its own
// test binary/process stops that mutation from corrupting this package's
// TestCommittedOpenAPIJSONMatchesGeneratedDocument, which asserts the widget
// routes still emit Huma's default RFC 9457 error schema.
func RegisterBenchRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "bench-plaintext",
		Method:      http.MethodGet,
		Path:        "/plaintext",
		Summary:     "Plaintext benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *struct{}) (*plaintextOutput, error) {
		return &plaintextOutput{
			ContentType: "text/plain; charset=utf-8",
			Body:        []byte("Hello, World!"),
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-json",
		Method:      http.MethodGet,
		Path:        "/json",
		Summary:     "JSON benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *struct{}) (*jsonMessageOutput, error) {
		return &jsonMessageOutput{
			Body: SuccessEnvelope[map[string]string]{Data: map[string]string{"message": "Hello, World!"}},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-get-user",
		Method:      http.MethodGet,
		Path:        "/users/{id}",
		Summary:     "Path parameter benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *getBenchUserInput) (*getBenchUserOutput, error) {
		return &getBenchUserOutput{
			Body: SuccessEnvelope[BenchUser]{Data: BenchUser{ID: input.ID, Name: "Ada Lovelace"}},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "bench-create-user",
		Method:      http.MethodPost,
		Path:        "/users",
		Summary:     "Validated POST benchmark scenario",
		Tags:        []string{"Benchmark"},
	}, func(ctx context.Context, input *createBenchUserInput) (*createBenchUserOutput, error) {
		return &createBenchUserOutput{
			Body: SuccessEnvelope[BenchUser]{
				Data: BenchUser{ID: "user-1", Name: input.Body.Name, Email: input.Body.Email},
			},
		}, nil
	})
}
