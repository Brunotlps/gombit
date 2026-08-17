package resourcegen

import (
	"fmt"
	"go/format"
	"strings"
)

type fileSpec struct {
	relPath string
	content []byte
	owned   bool // AST-edited user file; additive, not banner-gated
}

type renderContext struct {
	Resource   ResourceName
	Fields     []Field
	Module     string
	ImportPath string
	ModelSpec  string
	APIPrefix  string
	Service    bool
	Repo       bool
	DataType   string
}

func newRenderContext(module string, name ResourceName, fields []Field, apiPrefix string, service, repo bool) renderContext {
	return renderContext{
		Resource:   name,
		Fields:     fields,
		Module:     module,
		ImportPath: module + "/internal/" + name.Package,
		ModelSpec:  module + "/internal/" + name.Package + "." + name.TypeName,
		APIPrefix:  apiPrefix,
		Service:    service,
		Repo:       repo,
		DataType:   unexported(name.TypeName) + "Data",
	}
}

func unexported(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func renderFeatureFiles(ctx renderContext) ([]fileSpec, error) {
	files := []fileSpec{
		{relPath: fmt.Sprintf("internal/%s/%s.go", ctx.Resource.Package, ctx.Resource.FileBase), content: mustFormatGo(renderModel(ctx))},
		{relPath: fmt.Sprintf("internal/%s/handler.go", ctx.Resource.Package), content: mustFormatGo(renderHandler(ctx))},
		{relPath: fmt.Sprintf("internal/%s/routes.go", ctx.Resource.Package), content: mustFormatGo(renderRoutes(ctx))},
	}
	if ctx.Service {
		files = append(files, fileSpec{
			relPath: fmt.Sprintf("internal/%s/service.go", ctx.Resource.Package),
			content: mustFormatGo(renderService(ctx)),
		})
	}
	if ctx.Repo {
		files = append(files, fileSpec{
			relPath: fmt.Sprintf("internal/%s/repo.go", ctx.Resource.Package),
			content: mustFormatGo(renderRepo(ctx)),
		})
	}

	files = append(files,
		fileSpec{relPath: fmt.Sprintf("frontend/src/%s/list.ts", ctx.Resource.Package), content: []byte(renderListTS(ctx))},
		fileSpec{relPath: fmt.Sprintf("frontend/src/%s/form.ts", ctx.Resource.Package), content: []byte(renderFormTS(ctx))},
	)
	return files, nil
}

func mustFormatGo(src string) []byte {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		// Leave unformatted source so tests can show the parse error.
		return []byte(src + "\n// format error: " + err.Error() + "\n")
	}
	return formatted
}

func goBanner() string {
	return "// " + GeneratedBanner + "\n"
}

func tsBanner() string {
	return "/**\n * " + GeneratedBanner + "\n */\n"
}

func renderModel(ctx renderContext) string {
	var b strings.Builder
	b.WriteString(goBanner())
	b.WriteString("package ")
	b.WriteString(ctx.Resource.Package)
	b.WriteString("\n\nimport \"gorm.io/gorm\"\n\n")
	b.WriteString("// ")
	b.WriteString(ctx.Resource.TypeName)
	b.WriteString(" is the feature-package GORM model.\n")
	b.WriteString("type ")
	b.WriteString(ctx.Resource.TypeName)
	b.WriteString(" struct {\n\tgorm.Model\n")
	for _, field := range ctx.Fields {
		b.WriteByte('\t')
		b.WriteString(field.GoName)
		b.WriteByte(' ')
		b.WriteString(field.GoType)
		if tag := field.gormTag(); tag != "" {
			b.WriteString(" `gorm:\"")
			b.WriteString(tag)
			b.WriteString("\"`")
		}
		b.WriteByte('\n')
	}
	b.WriteString("}\n")
	return b.String()
}

func renderHandler(ctx renderContext) string {
	var b strings.Builder
	pkg := ctx.Resource.Package
	typ := ctx.Resource.TypeName
	data := ctx.DataType
	singular := strings.ToLower(typ)

	b.WriteString(goBanner())
	b.WriteString("package ")
	b.WriteString(pkg)
	b.WriteString("\n\nimport (\n\t\"context\"\n\t\"strconv\"\n\n")
	b.WriteString("\t\"github.com/LAA-Software-Engineering/gombit/contract\"\n")
	b.WriteString("\t\"gorm.io/gorm\"\n)\n\n")
	b.WriteString("// Handler serves " + pkg + " HTTP operations over GORM.\n")
	b.WriteString("type Handler struct {\n\tDB *gorm.DB\n}\n\n")
	b.WriteString("type " + data + " struct {\n")
	b.WriteString("\tID uint `json:\"id\" example:\"1\" doc:\"" + typ + " identifier\"`\n")
	for _, field := range ctx.Fields {
		b.WriteString("\t" + field.GoName + " " + field.GoType + " `" + field.jsonTag() + " doc:\"" + field.GoName + "\"`\n")
	}
	b.WriteString("}\n\n")
	listOut := "list" + ctx.Resource.Tag + "Output"
	b.WriteString("type " + listOut + " struct {\n")
	b.WriteString("\tBody contract.DataMeta[[]" + data + ", contract.PageMeta]\n}\n\n")
	b.WriteString("type get" + typ + "Input struct {\n")
	b.WriteString("\tID string `path:\"id\" doc:\"" + typ + " identifier\"`\n}\n\n")
	b.WriteString("type get" + typ + "Output struct {\n")
	b.WriteString("\tBody contract.Data[" + data + "]\n}\n\n")
	b.WriteString("type create" + typ + "Input struct {\n\tBody struct {\n")
	for _, field := range ctx.Fields {
		b.WriteString("\t\t" + field.GoName + " " + field.GoType + " `" + field.humaTags() + "`\n")
	}
	b.WriteString("\t}\n}\n\n")
	b.WriteString("type create" + typ + "Output struct {\n")
	b.WriteString("\tBody contract.Data[" + data + "]\n}\n\n")

	b.WriteString("func (h *Handler) list(ctx context.Context, _ *struct{}) (*" + listOut + ", error) {\n")
	b.WriteString("\tvar rows []" + typ + "\n")
	b.WriteString("\tif err := h.DB.WithContext(ctx).Order(\"id\").Find(&rows).Error; err != nil {\n")
	b.WriteString("\t\treturn nil, contract.WithContext(ctx, contract.Internal(\"list " + ctx.Resource.PluralSnake + "\"))\n")
	b.WriteString("\t}\n")
	b.WriteString("\titems := make([]" + data + ", 0, len(rows))\n")
	b.WriteString("\tfor _, row := range rows {\n\t\titems = append(items, to" + typ + "Data(row))\n\t}\n")
	b.WriteString("\treturn &" + listOut + "{\n")
	b.WriteString("\t\tBody: contract.DataMeta[[]" + data + ", contract.PageMeta]{\n")
	b.WriteString("\t\t\tData: items,\n")
	b.WriteString("\t\t\tMeta: &contract.PageMeta{Page: 1, PerPage: 20, Total: int64(len(items))},\n")
	b.WriteString("\t\t},\n")
	b.WriteString("\t}, nil\n}\n\n")

	b.WriteString("func (h *Handler) get(ctx context.Context, input *get" + typ + "Input) (*get" + typ + "Output, error) {\n")
	b.WriteString("\tid, err := strconv.ParseUint(input.ID, 10, 64)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, contract.WithContext(ctx, contract.NotFound(\"" + singular + " not found\"))\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvar row " + typ + "\n")
	b.WriteString("\tif err := h.DB.WithContext(ctx).First(&row, uint(id)).Error; err != nil {\n")
	b.WriteString("\t\treturn nil, contract.WithContext(ctx, contract.NotFound(\"" + singular + " not found\"))\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn &get" + typ + "Output{\n")
	b.WriteString("\t\tBody: contract.Data[" + data + "]{Data: to" + typ + "Data(row)},\n")
	b.WriteString("\t}, nil\n}\n\n")

	b.WriteString("func (h *Handler) create(ctx context.Context, input *create" + typ + "Input) (*create" + typ + "Output, error) {\n")
	b.WriteString("\trow := " + typ + "{\n")
	for _, field := range ctx.Fields {
		b.WriteString("\t\t" + field.GoName + ": input.Body." + field.GoName + ",\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("\tif err := h.DB.WithContext(ctx).Create(&row).Error; err != nil {\n")
	b.WriteString("\t\treturn nil, contract.WithContext(ctx, contract.Internal(\"create " + singular + "\"))\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn &create" + typ + "Output{\n")
	b.WriteString("\t\tBody: contract.Data[" + data + "]{Data: to" + typ + "Data(row)},\n")
	b.WriteString("\t}, nil\n}\n\n")

	b.WriteString("func to" + typ + "Data(row " + typ + ") " + data + " {\n")
	b.WriteString("\treturn " + data + "{ID: row.ID")
	for _, field := range ctx.Fields {
		b.WriteString(", " + field.GoName + ": row." + field.GoName)
	}
	b.WriteString("}\n}\n")
	return b.String()
}

func renderRoutes(ctx renderContext) string {
	var b strings.Builder
	pkg := ctx.Resource.Package
	typ := ctx.Resource.TypeName
	path := ctx.Resource.HTTPPath
	tag := ctx.Resource.Tag
	kebab := ctx.Resource.Kebab

	b.WriteString(goBanner())
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"net/http\"\n\n")
	b.WriteString("\t\"github.com/LAA-Software-Engineering/gombit/framework\"\n")
	b.WriteString("\t\"github.com/danielgtaylor/huma/v2\"\n)\n\n")
	b.WriteString("// Register mounts " + pkg + " Huma routes. Called explicitly from main; Gombit\n")
	b.WriteString("// does not discover feature packages by reflection.\n")
	b.WriteString("func Register(app *framework.App) {\n")
	b.WriteString("\th := &Handler{DB: app.DB()}\n")
	b.WriteString("\tprefix := app.Config().API.Prefix\n")
	b.WriteString("\tapi := app.API()\n\n")
	b.WriteString("\thuma.Register(api, huma.Operation{\n")
	b.WriteString("\t\tOperationID: \"list-" + kebab + "\",\n")
	b.WriteString("\t\tMethod:      http.MethodGet,\n")
	b.WriteString("\t\tPath:        prefix + \"" + path + "\",\n")
	b.WriteString("\t\tSummary:     \"List " + strings.ToLower(tag) + "\",\n")
	b.WriteString("\t\tTags:        []string{\"" + tag + "\"},\n")
	b.WriteString("\t}, h.list)\n\n")
	b.WriteString("\thuma.Register(api, huma.Operation{\n")
	b.WriteString("\t\tOperationID: \"get-" + pkg + "\",\n")
	b.WriteString("\t\tMethod:      http.MethodGet,\n")
	b.WriteString("\t\tPath:        prefix + \"" + path + "/{id}\",\n")
	b.WriteString("\t\tSummary:     \"Get a " + strings.ToLower(typ) + "\",\n")
	b.WriteString("\t\tTags:        []string{\"" + tag + "\"},\n")
	b.WriteString("\t}, h.get)\n\n")
	b.WriteString("\thuma.Register(api, huma.Operation{\n")
	b.WriteString("\t\tOperationID: \"create-" + pkg + "\",\n")
	b.WriteString("\t\tMethod:      http.MethodPost,\n")
	b.WriteString("\t\tPath:        prefix + \"" + path + "\",\n")
	b.WriteString("\t\tSummary:     \"Create a " + strings.ToLower(typ) + "\",\n")
	b.WriteString("\t\tTags:        []string{\"" + tag + "\"},\n")
	b.WriteString("\t}, h.create)\n")
	b.WriteString("}\n")
	return b.String()
}

func renderService(ctx renderContext) string {
	typ := ctx.Resource.TypeName
	pkg := ctx.Resource.Package
	var b strings.Builder
	b.WriteString(goBanner())
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"context\"\n\n\t\"gorm.io/gorm\"\n)\n\n")
	b.WriteString("// Service is an opt-in pass-through over GORM (--service). The generated\n")
	b.WriteString("// handler stays thin over GORM; this type exists so the file compiles.\n")
	b.WriteString("type Service struct {\n\tDB *gorm.DB\n}\n\n")
	b.WriteString("func NewService(db *gorm.DB) *Service {\n\treturn &Service{DB: db}\n}\n\n")
	b.WriteString("func (s *Service) List(ctx context.Context) ([]" + typ + ", error) {\n")
	b.WriteString("\tvar rows []" + typ + "\n")
	b.WriteString("\terr := s.DB.WithContext(ctx).Order(\"id\").Find(&rows).Error\n")
	b.WriteString("\treturn rows, err\n}\n\n")
	b.WriteString("func (s *Service) Get(ctx context.Context, id uint) (" + typ + ", error) {\n")
	b.WriteString("\tvar row " + typ + "\n")
	b.WriteString("\terr := s.DB.WithContext(ctx).First(&row, id).Error\n")
	b.WriteString("\treturn row, err\n}\n\n")
	b.WriteString("func (s *Service) Create(ctx context.Context, row *" + typ + ") error {\n")
	b.WriteString("\treturn s.DB.WithContext(ctx).Create(row).Error\n}\n")
	return b.String()
}

func renderRepo(ctx renderContext) string {
	typ := ctx.Resource.TypeName
	pkg := ctx.Resource.Package
	var b strings.Builder
	b.WriteString(goBanner())
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"context\"\n\n\t\"gorm.io/gorm\"\n)\n\n")
	b.WriteString("// Repo is an opt-in pass-through over GORM (--repo). Prefer the runtime\n")
	b.WriteString("// repository.New[T] helper instead of growing this file (D9).\n")
	b.WriteString("type Repo struct {\n\tDB *gorm.DB\n}\n\n")
	b.WriteString("func NewRepo(db *gorm.DB) *Repo {\n\treturn &Repo{DB: db}\n}\n\n")
	b.WriteString("func (r *Repo) List(ctx context.Context) ([]" + typ + ", error) {\n")
	b.WriteString("\tvar rows []" + typ + "\n")
	b.WriteString("\terr := r.DB.WithContext(ctx).Order(\"id\").Find(&rows).Error\n")
	b.WriteString("\treturn rows, err\n}\n\n")
	b.WriteString("func (r *Repo) Get(ctx context.Context, id uint) (" + typ + ", error) {\n")
	b.WriteString("\tvar row " + typ + "\n")
	b.WriteString("\terr := r.DB.WithContext(ctx).First(&row, id).Error\n")
	b.WriteString("\treturn row, err\n}\n\n")
	b.WriteString("func (r *Repo) Create(ctx context.Context, row *" + typ + ") error {\n")
	b.WriteString("\treturn r.DB.WithContext(ctx).Create(row).Error\n}\n")
	return b.String()
}

func jsIdent(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func renderListTS(ctx renderContext) string {
	labels := `["id"`
	values := "[(row as { id?: unknown }).id"
	for _, field := range ctx.Fields {
		labels += `, "` + field.JSONName + `"`
		values += ", (row as { " + field.JSONName + "?: unknown })." + field.JSONName
	}
	labels += "]"
	values += "]"

	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\nimport type { paths } from \"../api/generated/schema\";\n\n")
	b.WriteString("const apiBase = import.meta.env.VITE_API_URL || \"" + ctx.APIPrefix + "\";\n\n")
	b.WriteString("type ListResponse =\n")
	b.WriteString("  paths[\"" + ctx.APIPrefix + ctx.Resource.HTTPPath + "\"][\"get\"][\"responses\"][200][\"content\"][\"application/json\"];\n\n")
	b.WriteString("/**\n * Vanilla list/table page. Types come from the generated OpenAPI client\n")
	b.WriteString(" * (gombit client generate / gombit dev). Do not duplicate API DTOs here.\n */\n")
	b.WriteString("export async function render" + ctx.Resource.TypeName + "List(root: HTMLElement): Promise<void> {\n")
	b.WriteString("  root.replaceChildren();\n")
	b.WriteString("  const heading = document.createElement(\"h1\");\n")
	b.WriteString("  heading.textContent = \"" + ctx.Resource.Tag + "\";\n")
	b.WriteString("  root.append(heading);\n\n")
	b.WriteString("  const nav = document.createElement(\"p\");\n")
	b.WriteString("  const home = document.createElement(\"a\");\n")
	b.WriteString("  home.href = \"?\";\n")
	b.WriteString("  home.textContent = \"All resources\";\n")
	b.WriteString("  const create = document.createElement(\"a\");\n")
	b.WriteString("  create.href = \"?resource=" + ctx.Resource.Package + "&view=new\";\n")
	b.WriteString("  create.textContent = \"New " + ctx.Resource.TypeName + "\";\n")
	b.WriteString("  nav.append(home, document.createTextNode(\" · \"), create);\n")
	b.WriteString("  root.append(nav);\n\n")
	b.WriteString("  const table = document.createElement(\"table\");\n")
	b.WriteString("  const thead = document.createElement(\"thead\");\n")
	b.WriteString("  const headerRow = document.createElement(\"tr\");\n")
	b.WriteString("  for (const label of " + labels + ") {\n")
	b.WriteString("    const th = document.createElement(\"th\");\n")
	b.WriteString("    th.textContent = label;\n")
	b.WriteString("    headerRow.append(th);\n")
	b.WriteString("  }\n")
	b.WriteString("  thead.append(headerRow);\n")
	b.WriteString("  table.append(thead);\n")
	b.WriteString("  const tbody = document.createElement(\"tbody\");\n")
	b.WriteString("  table.append(tbody);\n")
	b.WriteString("  root.append(table);\n\n")
	b.WriteString("  const status = document.createElement(\"p\");\n")
	b.WriteString("  root.append(status);\n\n")
	b.WriteString("  try {\n")
	b.WriteString("    const response = await fetch(`${apiBase}" + ctx.Resource.HTTPPath + "`);\n")
	b.WriteString("    const body = (await response.json()) as ListResponse;\n")
	b.WriteString("    const rows = Array.isArray(body.data) ? body.data : [];\n")
	b.WriteString("    if (rows.length === 0) {\n")
	b.WriteString("      status.textContent = \"No " + ctx.Resource.Kebab + " yet.\";\n")
	b.WriteString("      return;\n")
	b.WriteString("    }\n")
	b.WriteString("    for (const row of rows) {\n")
	b.WriteString("      const tr = document.createElement(\"tr\");\n")
	b.WriteString("      const values: unknown[] = " + values + ";\n")
	b.WriteString("      for (const value of values) {\n")
	b.WriteString("        const td = document.createElement(\"td\");\n")
	b.WriteString("        td.textContent = value == null ? \"\" : String(value);\n")
	b.WriteString("        tr.append(td);\n")
	b.WriteString("      }\n")
	b.WriteString("      tbody.append(tr);\n")
	b.WriteString("    }\n")
	b.WriteString("  } catch (err: unknown) {\n")
	b.WriteString("    const message = err instanceof Error ? err.message : \"request failed\";\n")
	b.WriteString("    status.textContent = message;\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

func renderFormTS(ctx renderContext) string {
	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\nimport type { paths } from \"../api/generated/schema\";\n\n")
	b.WriteString("const apiBase = import.meta.env.VITE_API_URL || \"" + ctx.APIPrefix + "\";\n\n")
	b.WriteString("type CreateBody =\n")
	b.WriteString("  paths[\"" + ctx.APIPrefix + ctx.Resource.HTTPPath + "\"][\"post\"][\"requestBody\"][\"content\"][\"application/json\"];\n\n")
	b.WriteString("/**\n * Vanilla create form. Request/response types come from the generated\n")
	b.WriteString(" * OpenAPI client. Run gombit client generate or gombit dev after adding\n")
	b.WriteString(" * routes so frontend/src/api/generated exists.\n */\n")
	b.WriteString("export function render" + ctx.Resource.TypeName + "Form(root: HTMLElement): void {\n")
	b.WriteString("  root.replaceChildren();\n")
	b.WriteString("  const heading = document.createElement(\"h1\");\n")
	b.WriteString("  heading.textContent = \"New " + ctx.Resource.TypeName + "\";\n")
	b.WriteString("  root.append(heading);\n\n")
	b.WriteString("  const back = document.createElement(\"p\");\n")
	b.WriteString("  const link = document.createElement(\"a\");\n")
	b.WriteString("  link.href = \"?resource=" + ctx.Resource.Package + "\";\n")
	b.WriteString("  link.textContent = \"Back to list\";\n")
	b.WriteString("  back.append(link);\n")
	b.WriteString("  root.append(back);\n\n")
	b.WriteString("  const form = document.createElement(\"form\");\n")

	for _, field := range ctx.Fields {
		ident := jsIdent(field.JSONName)
		elem := "input"
		if field.Type == FieldText {
			elem = "textarea"
		}
		b.WriteString("  const " + ident + "Label = document.createElement(\"label\");\n")
		b.WriteString("  " + ident + "Label.textContent = \"" + field.GoName + "\";\n")
		b.WriteString("  const " + ident + "Input = document.createElement(\"" + elem + "\");\n")
		if field.Type != FieldText {
			b.WriteString("  " + ident + "Input.setAttribute(\"type\", \"" + field.inputKind() + "\");\n")
		}
		b.WriteString("  " + ident + "Input.setAttribute(\"name\", \"" + field.JSONName + "\");\n")
		if field.Required {
			b.WriteString("  " + ident + "Input.setAttribute(\"required\", \"true\");\n")
		}
		b.WriteString("  " + ident + "Label.append(" + ident + "Input);\n")
		b.WriteString("  form.append(" + ident + "Label);\n")
	}

	b.WriteString("  const submit = document.createElement(\"button\");\n")
	b.WriteString("  submit.type = \"submit\";\n")
	b.WriteString("  submit.textContent = \"Create\";\n")
	b.WriteString("  form.append(submit);\n")
	b.WriteString("  const status = document.createElement(\"p\");\n")
	b.WriteString("  root.append(form, status);\n\n")
	b.WriteString("  form.addEventListener(\"submit\", (event) => {\n")
	b.WriteString("    event.preventDefault();\n")
	b.WriteString("    const payload = {")
	for i, field := range ctx.Fields {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(" " + field.JSONName + ": read" + field.GoName + "()")
	}
	b.WriteString(" } as CreateBody;\n")
	b.WriteString("    void fetch(`${apiBase}" + ctx.Resource.HTTPPath + "`, {\n")
	b.WriteString("      method: \"POST\",\n")
	b.WriteString("      headers: { \"Content-Type\": \"application/json\" },\n")
	b.WriteString("      body: JSON.stringify(payload),\n")
	b.WriteString("    })\n")
	b.WriteString("      .then(async (response) => {\n")
	b.WriteString("        if (!response.ok) {\n")
	b.WriteString("          status.textContent = `create failed (${response.status})`;\n")
	b.WriteString("          return;\n")
	b.WriteString("        }\n")
	b.WriteString("        window.location.search = \"?resource=" + ctx.Resource.Package + "\";\n")
	b.WriteString("      })\n")
	b.WriteString("      .catch((err: unknown) => {\n")
	b.WriteString("        status.textContent = err instanceof Error ? err.message : \"request failed\";\n")
	b.WriteString("      });\n")
	b.WriteString("  });\n")

	for _, field := range ctx.Fields {
		ident := jsIdent(field.JSONName)
		tsType := "string"
		readExpr := "(" + ident + "Input as HTMLInputElement).value"
		switch field.Type {
		case FieldBool:
			tsType = "boolean"
			readExpr = "(" + ident + "Input as HTMLInputElement).checked"
		case FieldInt, FieldInt64, FieldUint:
			tsType = "number"
			readExpr = "Number((" + ident + "Input as HTMLInputElement).value)"
		case FieldText:
			readExpr = "(" + ident + "Input as HTMLTextAreaElement).value"
		}
		b.WriteString("  function read" + field.GoName + "(): " + tsType + " {\n")
		b.WriteString("    return " + readExpr + ";\n")
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	return b.String()
}
