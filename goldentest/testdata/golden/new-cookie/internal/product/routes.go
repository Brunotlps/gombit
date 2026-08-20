package product

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gombit-dev/gombit/framework"
)

// Register mounts product Huma routes. Called explicitly from main; Gombit
// does not discover feature packages by reflection.
func Register(app *framework.App) {
	h := &Handler{DB: app.DB()}
	prefix := app.Config().API.Prefix
	api := app.API()

	huma.Register(api, huma.Operation{
		OperationID: "list-products",
		Method:      http.MethodGet,
		Path:        prefix + "/products",
		Summary:     "List products",
		Tags:        []string{"Products"},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "get-product",
		Method:      http.MethodGet,
		Path:        prefix + "/products/{id}",
		Summary:     "Get a product",
		Tags:        []string{"Products"},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "create-product",
		Method:      http.MethodPost,
		Path:        prefix + "/products",
		Summary:     "Create a product",
		Tags:        []string{"Products"},
	}, h.create)
}
