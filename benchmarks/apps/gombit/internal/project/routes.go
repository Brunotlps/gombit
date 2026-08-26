package project

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gombit-dev/gombit/framework"
)

// Register mounts the canonical /api/projects Huma routes. Called
// explicitly from main; Gombit does not discover feature packages by
// reflection.
func Register(app *framework.App) {
	h := &Handler{DB: app.DB()}
	prefix := app.Config().API.Prefix
	api := app.API()

	huma.Register(api, huma.Operation{
		OperationID: "list-projects",
		Method:      http.MethodGet,
		Path:        prefix + "/projects",
		Summary:     "List projects",
		Tags:        []string{"Projects"},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "get-project",
		Method:      http.MethodGet,
		Path:        prefix + "/projects/{id}",
		Summary:     "Get a project",
		Tags:        []string{"Projects"},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "create-project",
		Method:      http.MethodPost,
		Path:        prefix + "/projects",
		Summary:     "Create a project",
		// Huma defaults POST to 200; without this it would silently
		// diverge from benchmarks/apps/gin-gorm's 201 Created.
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Projects"},
	}, h.create)

	huma.Register(api, huma.Operation{
		OperationID: "update-project",
		Method:      http.MethodPatch,
		Path:        prefix + "/projects/{id}",
		Summary:     "Update a project",
		Tags:        []string{"Projects"},
	}, h.update)

	huma.Register(api, huma.Operation{
		OperationID: "delete-project",
		Method:      http.MethodDelete,
		Path:        prefix + "/projects/{id}",
		Summary:     "Delete a project",
		Tags:        []string{"Projects"},
	}, h.delete)
}
