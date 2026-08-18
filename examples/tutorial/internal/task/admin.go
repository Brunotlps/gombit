package task

import (
	"github.com/LAA-Software-Engineering/gombit/admin"
	"github.com/LAA-Software-Engineering/gombit/framework"
)

// RegisterAdmin registers Task on the runtime admin. Feature packages own
// this call — the framework never discovers GORM models by itself (ADR-013).
//
// Register requires cookie auth; it returns an error under Bearer-only mode.
func RegisterAdmin(app *framework.App) error {
	return admin.Register(app, Task{}, admin.Options{
		Slug:     "tasks",
		Singular: "Task",
		Plural:   "Tasks",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "title", Type: admin.TypeString, Required: true},
			{Name: "done", Type: admin.TypeBoolean},
		},
		List:     []string{"title", "done"},
		Search:   []string{"title"},
		Ordering: []string{"title", "created_at"},
		Actions: admin.Actions{
			List: true, Detail: true, Create: true, Update: true, Delete: true,
		},
	})
}
