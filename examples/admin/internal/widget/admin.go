package widget

import (
	"github.com/gombit-dev/gombit/admin"
	"github.com/gombit-dev/gombit/framework"
)

// RegisterAdmin registers Widget on the runtime admin. Feature packages own
// this call — the framework never discovers GORM models by itself (ADR-013).
func RegisterAdmin(app *framework.App) error {
	return admin.Register(app, Widget{}, admin.Options{
		Slug:     "widgets",
		Singular: "Widget",
		Plural:   "Widgets",
		Fields: []admin.Field{
			{Name: "id", Type: admin.TypeInteger, ReadOnly: true},
			{Name: "name", Type: admin.TypeString, Required: true},
			{Name: "sku", Type: admin.TypeString},
		},
		List:     []string{"name", "sku"},
		Search:   []string{"name", "sku"},
		Ordering: []string{"name", "created_at"},
		Actions: admin.Actions{
			List: true, Detail: true, Create: true, Update: true, Delete: true,
		},
	})
}
