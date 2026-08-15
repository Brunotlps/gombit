package contractspike

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

const openAPIPath = "/openapi"

// Server is the M0-2 spike surface that proves Huma can share a Gin router
// with raw Gin-only routes.
type Server struct {
	Router *gin.Engine
	API    huma.API
}

// SuccessEnvelope is the D10 success response shape used by the spike routes.
type SuccessEnvelope[T any] struct {
	Data T `json:"data"`
}

// Widget is the sample typed resource used by the contract spike.
type Widget struct {
	ID    string `json:"id" example:"widget-1" doc:"Stable widget identifier"`
	Name  string `json:"name" example:"First widget" doc:"Human-readable widget name"`
	Color string `json:"color,omitempty" example:"blue" doc:"Optional display color"`
}

// CreateWidgetBody is the typed JSON request body for POST /widgets.
type CreateWidgetBody struct {
	Name  string `json:"name" minLength:"1" maxLength:"80" example:"Second widget" doc:"Human-readable widget name"`
	Color string `json:"color,omitempty" maxLength:"30" example:"green" doc:"Optional display color"`
}

type createWidgetInput struct {
	Body CreateWidgetBody
}

type listWidgetsOutput struct {
	Body SuccessEnvelope[[]Widget]
}

type createWidgetOutput struct {
	Body SuccessEnvelope[Widget]
}

// WidgetStore is a concurrency-safe in-memory store for the spike handlers.
type WidgetStore struct {
	mu      sync.Mutex
	nextID  int
	widgets []Widget
}

// NewWidgetStore creates an in-memory widget store seeded with the provided widgets.
func NewWidgetStore(seed []Widget) *WidgetStore {
	copied := append([]Widget(nil), seed...)

	nextID := len(copied) + 1
	return &WidgetStore{
		nextID:  nextID,
		widgets: copied,
	}
}

// List returns a copy of the stored widgets.
func (s *WidgetStore) List() []Widget {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]Widget(nil), s.widgets...)
}

// Create stores a widget from the provided request body and returns it.
func (s *WidgetStore) Create(body CreateWidgetBody) Widget {
	s.mu.Lock()
	defer s.mu.Unlock()

	widget := Widget{
		ID:    "widget-" + strconv.Itoa(s.nextID),
		Name:  body.Name,
		Color: body.Color,
	}
	s.nextID++
	s.widgets = append(s.widgets, widget)

	return widget
}

// NewServer creates the default Huma-over-Gin spike server.
func NewServer() *Server {
	return NewServerWithStore(NewWidgetStore([]Widget{
		{ID: "widget-1", Name: "First widget", Color: "blue"},
	}))
}

// NewServerWithStore creates a spike server backed by the provided store.
func NewServerWithStore(store *WidgetStore) *Server {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := humagin.New(router, Config())

	registerWidgetRoutes(api, store)
	registerRawRoutes(router)

	return &Server{
		Router: router,
		API:    api,
	}
}

// Config returns the Huma API configuration used by the spike.
func Config() huma.Config {
	config := huma.DefaultConfig("Gombit Contract Spike", "0.0.0")
	config.OpenAPIPath = openAPIPath
	config.DocsPath = ""
	config.SchemasPath = ""
	return config
}

// OpenAPIJSON marshals the Huma-generated OpenAPI document.
func OpenAPIJSON(api huma.API) ([]byte, error) {
	return json.MarshalIndent(api.OpenAPI(), "", "  ")
}

// WriteOpenAPI writes the Huma-generated OpenAPI document to path.
func WriteOpenAPI(path string) error {
	spec, err := OpenAPIJSON(NewServer().API)
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(spec, '\n'), 0o644)
}

func registerWidgetRoutes(api huma.API, store *WidgetStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-widgets",
		Method:      http.MethodGet,
		Path:        "/widgets",
		Summary:     "List widgets",
		Tags:        []string{"Widgets"},
	}, func(ctx context.Context, input *struct{}) (*listWidgetsOutput, error) {
		return &listWidgetsOutput{
			Body: SuccessEnvelope[[]Widget]{
				Data: store.List(),
			},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-widget",
		Method:      http.MethodPost,
		Path:        "/widgets",
		Summary:     "Create a widget",
		Tags:        []string{"Widgets"},
	}, func(ctx context.Context, input *createWidgetInput) (*createWidgetOutput, error) {
		return &createWidgetOutput{
			Body: SuccessEnvelope[Widget]{
				Data: store.Create(input.Body),
			},
		}, nil
	})
}

func registerRawRoutes(router *gin.Engine) {
	router.GET("/raw/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status": "ok",
			},
		})
	})
}
