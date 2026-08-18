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
	UI         string
	Service    bool
	Repo       bool
	DataType   string
}

func newRenderContext(module string, name ResourceName, fields []Field, apiPrefix, ui string, service, repo bool) renderContext {
	if ui == "" {
		ui = defaultUI
	}
	return renderContext{
		Resource:   name,
		Fields:     fields,
		Module:     module,
		ImportPath: module + "/internal/" + name.Package,
		ModelSpec:  module + "/internal/" + name.Package + "." + name.TypeName,
		APIPrefix:  apiPrefix,
		UI:         ui,
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
		fileSpec{relPath: fmt.Sprintf("frontend/src/%s/list.tsx", ctx.Resource.Package), content: []byte(renderListTSX(ctx))},
		fileSpec{relPath: fmt.Sprintf("frontend/src/%s/form.tsx", ctx.Resource.Package), content: []byte(renderFormTSX(ctx))},
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

func renderListTSX(ctx renderContext) string {
	if ctx.UI == "mui" {
		return renderMUIListTSX(ctx)
	}
	return renderMinimalListTSX(ctx)
}

func renderMinimalListTSX(ctx renderContext) string {
	listPath := ctx.APIPrefix + ctx.Resource.HTTPPath
	labels := `["id"`
	for _, field := range ctx.Fields {
		labels += `, "` + field.JSONName + `"`
	}
	labels += "]"

	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\nimport { useEffect, useState } from \"react\";\n")
	b.WriteString("import { Link } from \"react-router\";\n\n")
	b.WriteString("import { useApiClient } from \"../api/client\";\n")
	b.WriteString("import { unwrap } from \"../api/generated/client\";\n")
	b.WriteString("import type { paths } from \"../api/generated/schema\";\n\n")
	b.WriteString("const listPath = \"" + listPath + "\" as const;\n\n")
	b.WriteString("type ListResponse =\n")
	b.WriteString("  paths[typeof listPath][\"get\"][\"responses\"][200][\"content\"][\"application/json\"];\n")
	b.WriteString("type ListRow = NonNullable<ListResponse[\"data\"]>[number];\n\n")
	b.WriteString("/**\n * React list/table page. Types come from the generated OpenAPI client\n")
	b.WriteString(" * (gombit client generate / gombit dev). Do not duplicate API DTOs here.\n */\n")
	b.WriteString("export function " + ctx.Resource.TypeName + "ListPage() {\n")
	b.WriteString("  const client = useApiClient();\n")
	b.WriteString("  const [rows, setRows] = useState<ListRow[]>([]);\n")
	b.WriteString("  const [status, setStatus] = useState(\"Loading…\");\n\n")
	b.WriteString("  useEffect(() => {\n")
	b.WriteString("    let cancelled = false;\n")
	b.WriteString("    void (async () => {\n")
	b.WriteString("      try {\n")
	b.WriteString("        const listed = await unwrap(await client.GET(listPath));\n")
	b.WriteString("        if (cancelled) {\n")
	b.WriteString("          return;\n")
	b.WriteString("        }\n")
	b.WriteString("        const data = Array.isArray(listed.data) ? listed.data : [];\n")
	b.WriteString("        setRows(data);\n")
	b.WriteString("        setStatus(data.length === 0 ? \"No " + ctx.Resource.Kebab + " yet.\" : \"\");\n")
	b.WriteString("      } catch (err: unknown) {\n")
	b.WriteString("        if (cancelled) {\n")
	b.WriteString("          return;\n")
	b.WriteString("        }\n")
	b.WriteString("        setStatus(err instanceof Error ? err.message : \"request failed\");\n")
	b.WriteString("      }\n")
	b.WriteString("    })();\n")
	b.WriteString("    return () => {\n")
	b.WriteString("      cancelled = true;\n")
	b.WriteString("    };\n")
	b.WriteString("  }, [client]);\n\n")
	b.WriteString("  return (\n")
	b.WriteString("    <section>\n")
	b.WriteString("      <h1>" + ctx.Resource.Tag + "</h1>\n")
	b.WriteString("      <p>\n")
	b.WriteString("        <Link to=\"/\">Products</Link>\n")
	b.WriteString("        {\" · \"}\n")
	b.WriteString("        <Link to=\"/" + ctx.Resource.Kebab + "/new\">New " + ctx.Resource.TypeName + "</Link>\n")
	b.WriteString("      </p>\n")
	b.WriteString("      <table>\n")
	b.WriteString("        <thead>\n")
	b.WriteString("          <tr>\n")
	b.WriteString("            {" + labels + ".map((label) => (\n")
	b.WriteString("              <th key={label}>{label}</th>\n")
	b.WriteString("            ))}\n")
	b.WriteString("          </tr>\n")
	b.WriteString("        </thead>\n")
	b.WriteString("        <tbody>\n")
	b.WriteString("          {rows.map((row, index) => {\n")
	b.WriteString("            const record = row as { id?: unknown")
	for _, field := range ctx.Fields {
		b.WriteString("; " + field.JSONName + "?: unknown")
	}
	b.WriteString(" };\n")
	b.WriteString("            const values: unknown[] = [record.id")
	for _, field := range ctx.Fields {
		b.WriteString(", record." + field.JSONName)
	}
	b.WriteString("];\n")
	b.WriteString("            const key = record.id == null ? String(index) : String(record.id);\n")
	b.WriteString("            return (\n")
	b.WriteString("              <tr key={key}>\n")
	b.WriteString("                {values.map((value, cell) => (\n")
	b.WriteString("                  <td key={cell}>{value == null ? \"\" : String(value)}</td>\n")
	b.WriteString("                ))}\n")
	b.WriteString("              </tr>\n")
	b.WriteString("            );\n")
	b.WriteString("          })}\n")
	b.WriteString("        </tbody>\n")
	b.WriteString("      </table>\n")
	b.WriteString("      {status ? <p>{status}</p> : null}\n")
	b.WriteString("    </section>\n")
	b.WriteString("  );\n")
	b.WriteString("}\n")
	return b.String()
}

func renderFormTSX(ctx renderContext) string {
	if ctx.UI == "mui" {
		return renderMUIFormTSX(ctx)
	}
	return renderMinimalFormTSX(ctx)
}

func renderMinimalFormTSX(ctx renderContext) string {
	createPath := ctx.APIPrefix + ctx.Resource.HTTPPath
	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\nimport { useState } from \"react\";\n")
	b.WriteString("import { useForm } from \"react-hook-form\";\n")
	b.WriteString("import { Link, useNavigate } from \"react-router\";\n\n")
	b.WriteString("import { useApiClient } from \"../api/client\";\n")
	b.WriteString("import { applyContractErrors } from \"../api/formErrors\";\n")
	b.WriteString("import { unwrap } from \"../api/generated/client\";\n")
	b.WriteString("import type { paths } from \"../api/generated/schema\";\n\n")
	b.WriteString("const createPath = \"" + createPath + "\" as const;\n\n")
	b.WriteString("type CreateBody =\n")
	b.WriteString("  paths[typeof createPath][\"post\"][\"requestBody\"][\"content\"][\"application/json\"];\n\n")
	b.WriteString("type FormValues = {\n")
	for _, field := range ctx.Fields {
		b.WriteString("  " + field.JSONName + ": " + tsFormType(field) + ";\n")
	}
	b.WriteString("};\n\n")
	b.WriteString("/**\n * React Hook Form create page. Request/response types come from the\n")
	b.WriteString(" * generated OpenAPI client. D10 error.fields map through applyContractErrors.\n")
	b.WriteString(" * Run gombit client generate or gombit dev after adding routes.\n */\n")
	b.WriteString("export function " + ctx.Resource.TypeName + "FormPage() {\n")
	b.WriteString("  const client = useApiClient();\n")
	b.WriteString("  const navigate = useNavigate();\n")
	b.WriteString("  const [status, setStatus] = useState(\"\");\n")
	b.WriteString("  const {\n")
	b.WriteString("    register,\n")
	b.WriteString("    handleSubmit,\n")
	b.WriteString("    setError,\n")
	b.WriteString("    formState: { errors, isSubmitting },\n")
	b.WriteString("  } = useForm<FormValues>({\n")
	b.WriteString("    defaultValues: {")
	for i, field := range ctx.Fields {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(" " + field.JSONName + ": " + tsDefaultValue(field))
	}
	b.WriteString(" },\n")
	b.WriteString("  });\n\n")
	b.WriteString("  async function onSubmit(values: FormValues) {\n")
	b.WriteString("    setStatus(\"\");\n")
	b.WriteString("    try {\n")
	b.WriteString("      await unwrap(await client.POST(createPath, { body: values as CreateBody }));\n")
	b.WriteString("      navigate(\"/" + ctx.Resource.Kebab + "\");\n")
	b.WriteString("    } catch (err: unknown) {\n")
	b.WriteString("      if (!applyContractErrors(setError, err)) {\n")
	b.WriteString("        setStatus(err instanceof Error ? err.message : \"request failed\");\n")
	b.WriteString("      }\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n\n")
	b.WriteString("  return (\n")
	b.WriteString("    <section>\n")
	b.WriteString("      <h1>New " + ctx.Resource.TypeName + "</h1>\n")
	b.WriteString("      <p>\n")
	b.WriteString("        <Link to=\"/" + ctx.Resource.Kebab + "\">Back to list</Link>\n")
	b.WriteString("      </p>\n")
	b.WriteString("      <form onSubmit={handleSubmit(onSubmit)}>\n")
	for _, field := range ctx.Fields {
		b.WriteString(renderFormField(field))
	}
	b.WriteString("        <button type=\"submit\" disabled={isSubmitting}>\n")
	b.WriteString("          Create\n")
	b.WriteString("        </button>\n")
	b.WriteString("      </form>\n")
	b.WriteString("      {status ? <p>{status}</p> : null}\n")
	b.WriteString("    </section>\n")
	b.WriteString("  );\n")
	b.WriteString("}\n")
	return b.String()
}

func tsFormType(field Field) string {
	switch field.Type {
	case FieldBool:
		return "boolean"
	case FieldInt, FieldInt64, FieldUint:
		return "number"
	default:
		return "string"
	}
}

func tsDefaultValue(field Field) string {
	switch field.Type {
	case FieldBool:
		return "false"
	case FieldInt, FieldInt64, FieldUint:
		return "0"
	default:
		return `""`
	}
}

func renderFormField(field Field) string {
	ident := jsIdent(field.JSONName)
	var b strings.Builder
	b.WriteString("        <label>\n")
	b.WriteString("          " + field.GoName + "\n")
	switch field.Type {
	case FieldText:
		b.WriteString("          <textarea {...register(\"" + field.JSONName + "\"")
		if field.Required {
			b.WriteString(", { required: true }")
		}
		b.WriteString(")} />\n")
	case FieldBool:
		b.WriteString("          <input type=\"checkbox\" {...register(\"" + field.JSONName + "\")} />\n")
	case FieldInt, FieldInt64, FieldUint:
		b.WriteString("          <input type=\"number\" {...register(\"" + field.JSONName + "\", { valueAsNumber: true })} />\n")
	default:
		b.WriteString("          <input type=\"text\" {...register(\"" + field.JSONName + "\"")
		if field.Required {
			b.WriteString(", { required: true }")
		}
		b.WriteString(")} />\n")
	}
	b.WriteString("        </label>\n")
	b.WriteString("        {errors." + ident + "?.message ? <p>{errors." + ident + ".message}</p> : null}\n")
	return b.String()
}

func renderMUIListTSX(ctx renderContext) string {
	listPath := ctx.APIPrefix + ctx.Resource.HTTPPath
	colSpan := 1 + len(ctx.Fields)
	labels := `["id"`
	for _, field := range ctx.Fields {
		labels += `, "` + field.JSONName + `"`
	}
	labels += "]"

	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\nimport { useEffect, useState } from \"react\";\n")
	b.WriteString("import { Link } from \"react-router\";\n")
	b.WriteString("import AddIcon from \"@mui/icons-material/Add\";\n")
	b.WriteString("import {\n")
	b.WriteString("  Box,\n  Button,\n  CircularProgress,\n  Paper,\n")
	b.WriteString("  Table,\n  TableBody,\n  TableCell,\n  TableContainer,\n")
	b.WriteString("  TableHead,\n  TableRow,\n  Typography,\n")
	b.WriteString("} from \"@mui/material\";\n\n")
	b.WriteString("import { useApiClient } from \"../api/client\";\n")
	b.WriteString("import { unwrap } from \"../api/generated/client\";\n")
	b.WriteString("import type { paths } from \"../api/generated/schema\";\n\n")
	b.WriteString("const listPath = \"" + listPath + "\" as const;\n\n")
	b.WriteString("type ListResponse =\n")
	b.WriteString("  paths[typeof listPath][\"get\"][\"responses\"][200][\"content\"][\"application/json\"];\n")
	b.WriteString("type ListRow = NonNullable<ListResponse[\"data\"]>[number];\n\n")
	b.WriteString("/**\n * MUI Table list page. Types come from the generated OpenAPI client\n")
	b.WriteString(" * (gombit client generate / gombit dev). Do not duplicate API DTOs here.\n */\n")
	b.WriteString("export function " + ctx.Resource.TypeName + "ListPage() {\n")
	b.WriteString("  const client = useApiClient();\n")
	b.WriteString("  const [rows, setRows] = useState<ListRow[]>([]);\n")
	b.WriteString("  const [loading, setLoading] = useState(true);\n")
	b.WriteString("  const [status, setStatus] = useState(\"\");\n\n")
	b.WriteString("  useEffect(() => {\n")
	b.WriteString("    let cancelled = false;\n")
	b.WriteString("    void (async () => {\n")
	b.WriteString("      try {\n")
	b.WriteString("        const listed = await unwrap(await client.GET(listPath));\n")
	b.WriteString("        if (cancelled) {\n")
	b.WriteString("          return;\n")
	b.WriteString("        }\n")
	b.WriteString("        const data = Array.isArray(listed.data) ? listed.data : [];\n")
	b.WriteString("        setRows(data);\n")
	b.WriteString("        setStatus(data.length === 0 ? \"No " + ctx.Resource.Kebab + " yet.\" : \"\");\n")
	b.WriteString("      } catch (err: unknown) {\n")
	b.WriteString("        if (cancelled) {\n")
	b.WriteString("          return;\n")
	b.WriteString("        }\n")
	b.WriteString("        setStatus(err instanceof Error ? err.message : \"request failed\");\n")
	b.WriteString("      } finally {\n")
	b.WriteString("        if (!cancelled) {\n")
	b.WriteString("          setLoading(false);\n")
	b.WriteString("        }\n")
	b.WriteString("      }\n")
	b.WriteString("    })();\n")
	b.WriteString("    return () => {\n")
	b.WriteString("      cancelled = true;\n")
	b.WriteString("    };\n")
	b.WriteString("  }, [client]);\n\n")
	b.WriteString("  return (\n")
	b.WriteString("    <Box>\n")
	b.WriteString("      <Box sx={{ display: \"flex\", justifyContent: \"space-between\", alignItems: \"center\", mb: 2 }}>\n")
	b.WriteString("        <Typography variant=\"h4\" component=\"h1\">\n")
	b.WriteString("          " + ctx.Resource.Tag + "\n")
	b.WriteString("        </Typography>\n")
	b.WriteString("        <Button variant=\"contained\" component={Link} to=\"/" + ctx.Resource.Kebab + "/new\" startIcon={<AddIcon />}>\n")
	b.WriteString("          New " + ctx.Resource.TypeName + "\n")
	b.WriteString("        </Button>\n")
	b.WriteString("      </Box>\n")
	b.WriteString("      {loading ? (\n")
	b.WriteString("        <Box sx={{ display: \"flex\", justifyContent: \"center\", py: 6 }}>\n")
	b.WriteString("          <CircularProgress />\n")
	b.WriteString("        </Box>\n")
	b.WriteString("      ) : (\n")
	b.WriteString("        <TableContainer component={Paper}>\n")
	b.WriteString("          <Table>\n")
	b.WriteString("            <TableHead>\n")
	b.WriteString("              <TableRow>\n")
	b.WriteString("                {" + labels + ".map((label) => (\n")
	b.WriteString("                  <TableCell key={label}>{label}</TableCell>\n")
	b.WriteString("                ))}\n")
	b.WriteString("              </TableRow>\n")
	b.WriteString("            </TableHead>\n")
	b.WriteString("            <TableBody>\n")
	b.WriteString("              {rows.length === 0 ? (\n")
	b.WriteString("                <TableRow>\n")
	b.WriteString("                  <TableCell colSpan={" + fmt.Sprintf("%d", colSpan) + "} align=\"center\">\n")
	b.WriteString("                    {status || \"No " + ctx.Resource.Kebab + " yet.\"}\n")
	b.WriteString("                  </TableCell>\n")
	b.WriteString("                </TableRow>\n")
	b.WriteString("              ) : (\n")
	b.WriteString("                rows.map((row, index) => {\n")
	b.WriteString("                  const record = row as { id?: unknown")
	for _, field := range ctx.Fields {
		b.WriteString("; " + field.JSONName + "?: unknown")
	}
	b.WriteString(" };\n")
	b.WriteString("                  const values: unknown[] = [record.id")
	for _, field := range ctx.Fields {
		b.WriteString(", record." + field.JSONName)
	}
	b.WriteString("];\n")
	b.WriteString("                  const key = record.id == null ? String(index) : String(record.id);\n")
	b.WriteString("                  return (\n")
	b.WriteString("                    <TableRow key={key}>\n")
	b.WriteString("                      {values.map((value, cell) => (\n")
	b.WriteString("                        <TableCell key={cell}>{value == null ? \"\" : String(value)}</TableCell>\n")
	b.WriteString("                      ))}\n")
	b.WriteString("                    </TableRow>\n")
	b.WriteString("                  );\n")
	b.WriteString("                })\n")
	b.WriteString("              )}\n")
	b.WriteString("            </TableBody>\n")
	b.WriteString("          </Table>\n")
	b.WriteString("        </TableContainer>\n")
	b.WriteString("      )}\n")
	b.WriteString("    </Box>\n")
	b.WriteString("  );\n")
	b.WriteString("}\n")
	return b.String()
}

func renderMUIFormTSX(ctx renderContext) string {
	createPath := ctx.APIPrefix + ctx.Resource.HTTPPath
	needsCheckbox := false
	for _, field := range ctx.Fields {
		if field.Type == FieldBool {
			needsCheckbox = true
			break
		}
	}

	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\nimport { useState } from \"react\";\n")
	b.WriteString("import { Controller, useForm } from \"react-hook-form\";\n")
	b.WriteString("import { Link, useNavigate } from \"react-router\";\n")
	b.WriteString("import { Alert, Box, Button, Paper, TextField, Typography")
	if needsCheckbox {
		b.WriteString(", Checkbox, FormControlLabel")
	}
	b.WriteString(" } from \"@mui/material\";\n\n")
	b.WriteString("import { useApiClient } from \"../api/client\";\n")
	b.WriteString("import { applyContractErrors } from \"../api/formErrors\";\n")
	b.WriteString("import { unwrap } from \"../api/generated/client\";\n")
	b.WriteString("import type { paths } from \"../api/generated/schema\";\n\n")
	b.WriteString("const createPath = \"" + createPath + "\" as const;\n\n")
	b.WriteString("type CreateBody =\n")
	b.WriteString("  paths[typeof createPath][\"post\"][\"requestBody\"][\"content\"][\"application/json\"];\n\n")
	b.WriteString("type FormValues = {\n")
	for _, field := range ctx.Fields {
		b.WriteString("  " + field.JSONName + ": " + tsFormType(field) + ";\n")
	}
	b.WriteString("};\n\n")
	b.WriteString("/**\n * MUI TextField create page. Request/response types come from the\n")
	b.WriteString(" * generated OpenAPI client. D10 error.fields map through applyContractErrors.\n")
	b.WriteString(" * Run gombit client generate or gombit dev after adding routes.\n */\n")
	b.WriteString("export function " + ctx.Resource.TypeName + "FormPage() {\n")
	b.WriteString("  const client = useApiClient();\n")
	b.WriteString("  const navigate = useNavigate();\n")
	b.WriteString("  const [status, setStatus] = useState(\"\");\n")
	b.WriteString("  const {\n")
	b.WriteString("    control,\n")
	b.WriteString("    handleSubmit,\n")
	b.WriteString("    setError,\n")
	b.WriteString("    formState: { isSubmitting },\n")
	b.WriteString("  } = useForm<FormValues>({\n")
	b.WriteString("    defaultValues: {")
	for i, field := range ctx.Fields {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(" " + field.JSONName + ": " + tsDefaultValue(field))
	}
	b.WriteString(" },\n")
	b.WriteString("  });\n\n")
	b.WriteString("  async function onSubmit(values: FormValues) {\n")
	b.WriteString("    setStatus(\"\");\n")
	b.WriteString("    try {\n")
	b.WriteString("      await unwrap(await client.POST(createPath, { body: values as CreateBody }));\n")
	b.WriteString("      navigate(\"/" + ctx.Resource.Kebab + "\");\n")
	b.WriteString("    } catch (err: unknown) {\n")
	b.WriteString("      if (!applyContractErrors(setError, err)) {\n")
	b.WriteString("        setStatus(err instanceof Error ? err.message : \"request failed\");\n")
	b.WriteString("      }\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n\n")
	b.WriteString("  return (\n")
	b.WriteString("    <Box>\n")
	b.WriteString("      <Typography variant=\"h4\" component=\"h1\" sx={{ mb: 1 }}>\n")
	b.WriteString("        New " + ctx.Resource.TypeName + "\n")
	b.WriteString("      </Typography>\n")
	b.WriteString("      <Button component={Link} to=\"/" + ctx.Resource.Kebab + "\" sx={{ mb: 2 }}>\n")
	b.WriteString("        Back to list\n")
	b.WriteString("      </Button>\n")
	b.WriteString("      <Paper sx={{ p: 3, maxWidth: 480 }}>\n")
	b.WriteString("        <Box component=\"form\" onSubmit={handleSubmit(onSubmit)} sx={{ display: \"flex\", flexDirection: \"column\", gap: 2 }}>\n")
	for _, field := range ctx.Fields {
		b.WriteString(renderMUIFormField(field))
	}
	b.WriteString("          <Button type=\"submit\" variant=\"contained\" disabled={isSubmitting}>\n")
	b.WriteString("            Create\n")
	b.WriteString("          </Button>\n")
	b.WriteString("        </Box>\n")
	b.WriteString("        {status ? (\n")
	b.WriteString("          <Alert severity=\"error\" sx={{ mt: 2 }}>\n")
	b.WriteString("            {status}\n")
	b.WriteString("          </Alert>\n")
	b.WriteString("        ) : null}\n")
	b.WriteString("      </Paper>\n")
	b.WriteString("    </Box>\n")
	b.WriteString("  );\n")
	b.WriteString("}\n")
	return b.String()
}

func renderMUIFormField(field Field) string {
	var b strings.Builder
	rules := ""
	if field.Required && field.Type != FieldBool {
		rules = " rules={{ required: true }}"
	}
	b.WriteString("          <Controller\n")
	b.WriteString("            name=\"" + field.JSONName + "\"\n")
	b.WriteString("            control={control}\n")
	if rules != "" {
		b.WriteString("           " + rules + "\n")
	}
	b.WriteString("            render={({ field, fieldState }) => (\n")
	switch field.Type {
	case FieldBool:
		b.WriteString("              <FormControlLabel\n")
		b.WriteString("                control={\n")
		b.WriteString("                  <Checkbox\n")
		b.WriteString("                    {...field}\n")
		b.WriteString("                    checked={Boolean(field.value)}\n")
		b.WriteString("                    disabled={isSubmitting}\n")
		b.WriteString("                  />\n")
		b.WriteString("                }\n")
		b.WriteString("                label=\"" + field.GoName + "\"\n")
		b.WriteString("              />\n")
	case FieldText:
		b.WriteString("              <TextField\n")
		b.WriteString("                {...field}\n")
		b.WriteString("                label=\"" + field.GoName + "\"\n")
		b.WriteString("                fullWidth\n")
		b.WriteString("                multiline\n")
		b.WriteString("                minRows={3}\n")
		b.WriteString("                error={!!fieldState.error}\n")
		b.WriteString("                helperText={fieldState.error?.message}\n")
		b.WriteString("                disabled={isSubmitting}\n")
		b.WriteString("              />\n")
	case FieldInt, FieldInt64, FieldUint:
		b.WriteString("              <TextField\n")
		b.WriteString("                {...field}\n")
		b.WriteString("                type=\"number\"\n")
		b.WriteString("                label=\"" + field.GoName + "\"\n")
		b.WriteString("                fullWidth\n")
		b.WriteString("                error={!!fieldState.error}\n")
		b.WriteString("                helperText={fieldState.error?.message}\n")
		b.WriteString("                disabled={isSubmitting}\n")
		b.WriteString("                onChange={(event) => {\n")
		b.WriteString("                  const raw = event.target.value;\n")
		b.WriteString("                  field.onChange(raw === \"\" ? 0 : Number(raw));\n")
		b.WriteString("                }}\n")
		b.WriteString("              />\n")
	default:
		b.WriteString("              <TextField\n")
		b.WriteString("                {...field}\n")
		b.WriteString("                label=\"" + field.GoName + "\"\n")
		b.WriteString("                fullWidth\n")
		b.WriteString("                error={!!fieldState.error}\n")
		b.WriteString("                helperText={fieldState.error?.message}\n")
		b.WriteString("                disabled={isSubmitting}\n")
		b.WriteString("              />\n")
	}
	b.WriteString("            )}\n")
	b.WriteString("          />\n")
	return b.String()
}
