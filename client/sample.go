package client

import (
	"context"
	"net/http"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/LAA-Software-Engineering/gombit/framework"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

type sampleWidget struct {
	ID    string `json:"id" example:"widget-1" doc:"Stable widget identifier"`
	Name  string `json:"name" example:"First widget" doc:"Human-readable widget name"`
	Color string `json:"color,omitempty" example:"blue" doc:"Optional display color"`
}

type sampleCreateBody struct {
	Name  string `json:"name" minLength:"1" maxLength:"80" example:"Second widget"`
	Color string `json:"color,omitempty" maxLength:"30" example:"green"`
}

// SampleApp returns a framework app with the contract-example widget routes.
// Tests and the client example use it as the sample OpenAPI source.
func SampleApp() (*framework.App, error) {
	gin.SetMode(gin.TestMode)
	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.HTTP.Addr = "127.0.0.1:0"
	app, err := framework.New(framework.WithConfig(cfg))
	if err != nil {
		return nil, err
	}
	registerSampleWidgetRoutes(app.API(), app.Config().API.Prefix)
	return app, nil
}

func registerSampleWidgetRoutes(api huma.API, prefix string) {
	type createInput struct {
		Body sampleCreateBody
	}
	type createOutput struct {
		Body contract.Data[sampleWidget]
	}
	type listOutput struct {
		Body contract.DataMeta[[]sampleWidget, contract.PageMeta]
	}
	type getInput struct {
		ID string `path:"id" doc:"Widget identifier"`
	}
	type getOutput struct {
		Body contract.Data[sampleWidget]
	}

	huma.Register(api, huma.Operation{
		OperationID: "list-widgets",
		Method:      http.MethodGet,
		Path:        prefix + "/widgets",
		Summary:     "List widgets",
	}, func(ctx context.Context, input *struct{}) (*listOutput, error) {
		return &listOutput{
			Body: contract.DataMeta[[]sampleWidget, contract.PageMeta]{
				Data: []sampleWidget{{ID: "widget-1", Name: "First widget", Color: "blue"}},
				Meta: &contract.PageMeta{Page: 1, PerPage: 20, Total: 1},
			},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-widget",
		Method:      http.MethodGet,
		Path:        prefix + "/widgets/{id}",
		Summary:     "Get a widget",
	}, func(ctx context.Context, input *getInput) (*getOutput, error) {
		if input.ID != "widget-1" {
			return nil, contract.WithContext(ctx, contract.NotFound("widget not found"))
		}
		return &getOutput{
			Body: contract.Data[sampleWidget]{
				Data: sampleWidget{ID: "widget-1", Name: "First widget", Color: "blue"},
			},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-widget",
		Method:      http.MethodPost,
		Path:        prefix + "/widgets",
		Summary:     "Create a widget",
	}, func(ctx context.Context, input *createInput) (*createOutput, error) {
		return &createOutput{
			Body: contract.Data[sampleWidget]{
				Data: sampleWidget{ID: "widget-1", Name: input.Body.Name, Color: input.Body.Color},
			},
		}, nil
	})
}
