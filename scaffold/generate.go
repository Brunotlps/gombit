package scaffold

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/LAA-Software-Engineering/gombit/config"
)

// Generate scaffolds a new Gombit application into opts.Dest (or <workdir>/<name>).
func Generate(ctx context.Context, opts Options) error {
	if ctx == nil {
		return errors.New("scaffold: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := opts.normalize(); err != nil {
		return err
	}
	if err := opts.resolveInteractive(); err != nil {
		return err
	}
	if err := opts.validate(); err != nil {
		return err
	}

	if err := checkDestination(opts); err != nil {
		return err
	}

	vars := templateVars{
		Name:           opts.Name,
		Module:         opts.Module,
		Database:       opts.Database,
		Cache:          opts.Cache,
		Auth:           opts.Auth,
		UI:             opts.UI,
		APIPrefix:      DefaultAPIPrefix,
		DatabaseDSN:    defaultDSN(opts.Database, opts.Name),
		CacheNamespace: config.DefaultCacheNamespace(opts.Name, config.EnvironmentDevelopment),
		GoVersion:      generatedGoVersion,
	}

	files, err := renderFiles(vars)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		full := filepath.Join(opts.Dest, filepath.FromSlash(file.relPath))
		action := "create"
		if _, err := os.Stat(full); err == nil {
			action = "modify"
		}
		display, err := displayPath(opts.WorkDir, full)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(opts.Stdout, "%s %s\n", action, display); err != nil {
			return err
		}
		if opts.DryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return fmt.Errorf("scaffold: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, file.content, 0o600); err != nil {
			return fmt.Errorf("scaffold: write %s: %w", display, err)
		}
	}
	return nil
}

type templateVars struct {
	Name           string
	Module         string
	Database       string
	Cache          string
	Auth           string
	UI             string
	APIPrefix      string
	DatabaseDSN    string
	CacheNamespace string
	GoVersion      string
}

type renderedFile struct {
	relPath string
	content []byte
}

func renderFiles(vars templateVars) ([]renderedFile, error) {
	var files []renderedFile
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		rel = strings.TrimSuffix(rel, ".tmpl")
		rel = rewriteTemplateName(rel)
		// #nosec G304 -- path is a rooted embed.FS entry from WalkDir
		raw, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("scaffold: read template %s: %w", path, err)
		}
		content, err := executeTemplate(path, string(raw), vars)
		if err != nil {
			return err
		}
		if strings.HasSuffix(rel, ".go") {
			formatted, err := format.Source(content)
			if err != nil {
				return fmt.Errorf("scaffold: format %s: %w", rel, err)
			}
			content = formatted
		}
		files = append(files, renderedFile{relPath: rel, content: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	files = append(files,
		renderedFile{relPath: "database/migrations/.gitkeep", content: []byte("")},
		renderedFile{relPath: "database/seeds/.gitkeep", content: []byte("")},
	)
	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})
	return files, nil
}

func rewriteTemplateName(rel string) string {
	switch rel {
	case "env.example":
		return ".env.example"
	case "gitignore":
		return ".gitignore"
	default:
		return rel
	}
}

func executeTemplate(name, src string, vars templateVars) ([]byte, error) {
	tmpl, err := template.New(name).Parse(src)
	if err != nil {
		return nil, fmt.Errorf("scaffold: parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("scaffold: execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func checkDestination(opts Options) error {
	info, err := os.Stat(opts.Dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("scaffold: stat dest: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("scaffold: destination %q exists and is not a directory", opts.Dest)
	}
	entries, err := os.ReadDir(opts.Dest)
	if err != nil {
		return fmt.Errorf("scaffold: read dest: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	if opts.Force {
		return nil
	}
	return fmt.Errorf("scaffold: destination %q is not empty; pass --force to overwrite", opts.Dest)
}

func defaultDSN(driver, name string) string {
	switch driver {
	case "postgres":
		return "postgres://USER:PASSWORD@127.0.0.1:5432/" + name + "?sslmode=disable"
	case "mysql":
		return "USER:PASSWORD@tcp(127.0.0.1:3306)/" + name + "?parseTime=true"
	default:
		return "file:gombit.db?cache=shared&_fk=1"
	}
}

func displayPath(workDir, full string) (string, error) {
	rel, err := filepath.Rel(workDir, full)
	if err != nil {
		return "", fmt.Errorf("scaffold: relative path: %w", err)
	}
	return filepath.ToSlash(rel), nil
}
