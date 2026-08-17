package resourcegen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/migrations"
)

// Generate writes a feature-package resource into an existing Gombit app.
func Generate(ctx context.Context, opts Options) error {
	if ctx == nil {
		return errors.New("resourcegen: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := opts.normalize(); err != nil {
		return err
	}
	if err := opts.validateAppLayout(); err != nil {
		return err
	}

	name, err := parseResourceName(opts.Name)
	if err != nil {
		return err
	}
	fields, err := parseFields(opts.Fields)
	if err != nil {
		return err
	}
	module, err := readModulePath(opts.WorkDir)
	if err != nil {
		return err
	}
	apiPrefix := readAPIPrefix(opts.WorkDir)
	ctxData := newRenderContext(module, name, fields, apiPrefix, opts.Service, opts.Repo)

	files, err := renderFeatureFiles(ctxData)
	if err != nil {
		return err
	}
	for i := range files {
		if strings.HasSuffix(files[i].relPath, ".go") {
			formatted, fmtErr := format.Source(files[i].content)
			if fmtErr != nil {
				return fmt.Errorf("resourcegen: format %s: %w", files[i].relPath, fmtErr)
			}
			files[i].content = formatted
		}
	}

	resources := collectFrontendResources(opts.WorkDir, name)
	files = append(files, fileSpec{
		relPath: resourcesTSRel,
		content: renderResourcesTS(resources, apiPrefix),
	})

	mainPath := filepath.Join(opts.WorkDir, filepath.FromSlash(serverMainRel))
	platformPath := filepath.Join(opts.WorkDir, filepath.FromSlash(platformDBRel))
	// #nosec G304 -- application files under the user work dir
	mainSrc, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("resourcegen: read %s: %w", serverMainRel, err)
	}
	// #nosec G304 -- application files under the user work dir
	platformSrc, err := os.ReadFile(platformPath)
	if err != nil {
		return fmt.Errorf("resourcegen: read %s: %w", platformDBRel, err)
	}
	newMain, err := AddImportAndRegister(mainSrc, ctxData.ImportPath, name.Package)
	if err != nil {
		return err
	}
	newPlatform, err := AddAutoMigrateModel(platformSrc, ctxData.ImportPath, name.Package, name.TypeName)
	if err != nil {
		return err
	}
	files = append(files,
		fileSpec{relPath: serverMainRel, content: newMain, owned: true},
		fileSpec{relPath: platformDBRel, content: newPlatform, owned: true},
	)

	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})

	planned, err := planWrites(opts, files)
	if err != nil {
		return err
	}

	for _, item := range planned {
		if _, err := fmt.Fprintf(opts.Stdout, "%s %s\n", item.action, item.display); err != nil {
			return err
		}
		if opts.DryRun || !item.write {
			continue
		}
		full := filepath.Join(opts.WorkDir, filepath.FromSlash(item.relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return fmt.Errorf("resourcegen: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, item.content, 0o644); err != nil { //nolint:gosec // generated source is a non-secret artifact
			return fmt.Errorf("resourcegen: write %s: %w", item.display, err)
		}
	}

	if opts.DryRun {
		return nil
	}
	if _, err := fmt.Fprintf(opts.Stdout, "GORM model is Atlas-loader ready: %s\n", ctxData.ModelSpec); err != nil {
		return err
	}
	return maybeMakeMigrations(ctx, opts, ctxData)
}

type plannedFile struct {
	relPath string
	display string
	content []byte
	action  string
	write   bool
}

func planWrites(opts Options, files []fileSpec) ([]plannedFile, error) {
	var planned []plannedFile
	for _, file := range files {
		full := filepath.Join(opts.WorkDir, filepath.FromSlash(file.relPath))
		display, err := displayPath(opts.WorkDir, full)
		if err != nil {
			return nil, err
		}
		existing, err := os.ReadFile(full) // #nosec G304 -- generator output path
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("resourcegen: read %s: %w", display, err)
		}
		exists := err == nil
		if exists && bytes.Equal(existing, file.content) {
			continue
		}
		if exists {
			if err := checkOverwrite(display, existing, file, opts.Force); err != nil {
				return nil, err
			}
		}
		action := "create"
		if exists {
			action = "modify"
		}
		planned = append(planned, plannedFile{
			relPath: file.relPath,
			display: display,
			content: file.content,
			action:  action,
			write:   true,
		})
	}
	return planned, nil
}

func checkOverwrite(display string, existing []byte, file fileSpec, force bool) error {
	if force {
		return nil
	}
	if file.owned {
		// Additive AST edits of known registration points.
		return nil
	}
	if file.relPath == resourcesTSRel && bytes.Contains(existing, []byte(GeneratedBanner)) {
		return nil
	}
	if bytes.Contains(existing, []byte(GeneratedBanner)) {
		return fmt.Errorf("resourcegen: refuse to overwrite %s without --force (generated file differs from this run)", display)
	}
	return fmt.Errorf("resourcegen: refuse to overwrite %s without --force (not generated by gombit)", display)
}

func readModulePath(workDir string) (string, error) {
	path := filepath.Join(workDir, "go.mod")
	// #nosec G304 -- go.mod inside the application work dir
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("resourcegen: read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if module == "" {
				break
			}
			return module, nil
		}
	}
	return "", errors.New("resourcegen: go.mod is missing a module path")
}

func readAPIPrefix(workDir string) string {
	path := filepath.Join(workDir, "gombit.yaml")
	// #nosec G304 -- optional project file
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultAPIPrefix
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "api_prefix:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "api_prefix:"))
		value = strings.Trim(value, `"'`)
		if value == "" {
			return defaultAPIPrefix
		}
		return value
	}
	return defaultAPIPrefix
}

func collectFrontendResources(workDir string, current ResourceName) []ResourceName {
	result := []ResourceName{current}
	seen := map[string]struct{}{current.Package: {}}
	dir := filepath.Join(workDir, "frontend", "src")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, ok := seen[entry.Name()]; ok {
			continue
		}
		listPath := filepath.Join(dir, entry.Name(), "list.ts")
		if _, err := os.Stat(listPath); err != nil {
			continue
		}
		parsed, err := parseResourceName(entry.Name())
		if err != nil {
			continue
		}
		result = append(result, parsed)
		seen[entry.Name()] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Package < result[j].Package
	})
	return result
}

func renderResourcesTS(resources []ResourceName, apiPrefix string) []byte {
	var b strings.Builder
	b.WriteString(tsBanner())
	b.WriteString("\n")
	b.WriteString("// Vanilla list/table + create form pages. Types are imported from\n")
	b.WriteString("// ./api/generated (gombit client generate / gombit dev). Full React\n")
	b.WriteString("// Router + React Hook Form is M5-1. API prefix: " + apiPrefix + "\n")
	b.WriteString("// Access tokens stay in memory; this file does not use localStorage.\n\n")
	for _, res := range resources {
		b.WriteString("import { render")
		b.WriteString(res.TypeName)
		b.WriteString("List } from \"./")
		b.WriteString(res.Package)
		b.WriteString("/list\";\n")
		b.WriteString("import { render")
		b.WriteString(res.TypeName)
		b.WriteString("Form } from \"./")
		b.WriteString(res.Package)
		b.WriteString("/form\";\n")
	}
	b.WriteString("\nexport type GeneratedResource = {\n")
	b.WriteString("  slug: string;\n  title: string;\n")
	b.WriteString("  mountList: (root: HTMLElement) => Promise<void> | void;\n")
	b.WriteString("  mountForm: (root: HTMLElement) => Promise<void> | void;\n")
	b.WriteString("};\n\n")
	b.WriteString("export const generatedResources: GeneratedResource[] = [\n")
	for _, res := range resources {
		b.WriteString("  {\n")
		b.WriteString("    slug: \"" + res.Package + "\",\n")
		b.WriteString("    title: \"" + res.TypeName + "\",\n")
		b.WriteString("    mountList: render" + res.TypeName + "List,\n")
		b.WriteString("    mountForm: render" + res.TypeName + "Form,\n")
		b.WriteString("  },\n")
	}
	b.WriteString("];\n\n")
	b.WriteString("export function mountIfResource(root: HTMLElement): boolean {\n")
	b.WriteString("  const params = new URLSearchParams(window.location.search);\n")
	b.WriteString("  const slug = params.get(\"resource\");\n")
	b.WriteString("  if (!slug) {\n")
	b.WriteString("    if (generatedResources.length === 0) {\n")
	b.WriteString("      return false;\n")
	b.WriteString("    }\n")
	b.WriteString("    renderResourceIndex(root);\n")
	b.WriteString("    return true;\n")
	b.WriteString("  }\n")
	b.WriteString("  const resource = generatedResources.find((item) => item.slug === slug);\n")
	b.WriteString("  if (!resource) {\n")
	b.WriteString("    return false;\n")
	b.WriteString("  }\n")
	b.WriteString("  if (params.get(\"view\") === \"new\") {\n")
	b.WriteString("    void resource.mountForm(root);\n")
	b.WriteString("  } else {\n")
	b.WriteString("    void resource.mountList(root);\n")
	b.WriteString("  }\n")
	b.WriteString("  return true;\n")
	b.WriteString("}\n\n")
	b.WriteString("function renderResourceIndex(root: HTMLElement): void {\n")
	b.WriteString("  root.replaceChildren();\n")
	b.WriteString("  const heading = document.createElement(\"h1\");\n")
	b.WriteString("  heading.textContent = \"Resources\";\n")
	b.WriteString("  root.append(heading);\n")
	b.WriteString("  const list = document.createElement(\"ul\");\n")
	b.WriteString("  for (const resource of generatedResources) {\n")
	b.WriteString("    const item = document.createElement(\"li\");\n")
	b.WriteString("    const link = document.createElement(\"a\");\n")
	b.WriteString("    link.href = `?resource=${resource.slug}`;\n")
	b.WriteString("    link.textContent = resource.title;\n")
	b.WriteString("    item.append(link);\n")
	b.WriteString("    list.append(item);\n")
	b.WriteString("  }\n")
	b.WriteString("  root.append(list);\n")
	b.WriteString("}\n")
	return []byte(b.String())
}

func maybeMakeMigrations(ctx context.Context, opts Options, spec renderContext) error {
	if opts.skipAtlas {
		return printMakemigrationsHint(opts, spec, "skipped in tests")
	}
	atlasPath, lookErr := lookPath(opts.AtlasBin)
	if lookErr != nil || atlasPath == "" {
		return printMakemigrationsHint(opts, spec, "atlas not on PATH")
	}

	driver := readDatabaseDriver(opts.WorkDir)
	err := makeMigrations(ctx, migrations.Options{
		WorkDir:      opts.WorkDir,
		Name:         "create_" + spec.Resource.PluralSnake,
		Driver:       driver,
		MigrationDir: "database/migrations",
		AtlasBinary:  opts.AtlasBin,
		Models: []migrations.Model{
			{ImportPath: spec.ImportPath, TypeName: spec.Resource.TypeName},
		},
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})
	if err != nil {
		return fmt.Errorf("resourcegen: makemigrations: %w", err)
	}
	return nil
}

func printMakemigrationsHint(opts Options, spec renderContext, reason string) error {
	_, err := fmt.Fprintf(
		opts.Stdout,
		"note: %s; run: gombit db makemigrations create_%s --model %s\n",
		reason,
		spec.Resource.PluralSnake,
		spec.ModelSpec,
	)
	return err
}

func readDatabaseDriver(workDir string) config.DatabaseDriver {
	path := filepath.Join(workDir, "gombit.yaml")
	// #nosec G304 -- optional project file
	data, err := os.ReadFile(path)
	if err != nil {
		return config.DatabaseDriverSQLite
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "database:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "database:"))
		value = strings.Trim(value, `"'`)
		switch value {
		case "postgres":
			return config.DatabaseDriverPostgres
		case "mysql":
			return config.DatabaseDriverMySQL
		default:
			return config.DatabaseDriverSQLite
		}
	}
	return config.DatabaseDriverSQLite
}

func displayPath(workDir, full string) (string, error) {
	rel, err := filepath.Rel(workDir, full)
	if err != nil {
		return "", fmt.Errorf("resourcegen: relative path: %w", err)
	}
	return filepath.ToSlash(rel), nil
}

var lookPath = func(name string) (string, error) {
	return execLookPath(name)
}

var makeMigrations = migrations.MakeMigrations
