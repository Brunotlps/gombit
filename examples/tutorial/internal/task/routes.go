package task

import (
	"net/http"

	"github.com/gombit-dev/gombit/framework"
	"github.com/danielgtaylor/huma/v2"
)

// Register mounts task Huma routes. Called explicitly from main; Gombit does
// not discover feature packages by reflection.
func Register(app *framework.App) {
	h := &Handler{DB: app.DB()}
	prefix := app.Config().API.Prefix
	api := app.API()

	huma.Register(api, huma.Operation{
		OperationID: "list-tasks",
		Method:      http.MethodGet,
		Path:        prefix + "/tasks",
		Summary:     "List tasks",
		Tags:        []string{"Tasks"},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "get-task",
		Method:      http.MethodGet,
		Path:        prefix + "/tasks/{id}",
		Summary:     "Get a task",
		Tags:        []string{"Tasks"},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "create-task",
		Method:      http.MethodPost,
		Path:        prefix + "/tasks",
		Summary:     "Create a task",
		Tags:        []string{"Tasks"},
	}, h.create)
}
