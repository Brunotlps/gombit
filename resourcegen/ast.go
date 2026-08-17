package resourcegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// AddImportAndRegister adds `importPath` and `<pkg>.Register(app)` next to an
// existing `*.Register(app)` call (the scaffold's product.Register) or, if
// none exists, immediately before `framework.Run(app)`. The edit is
// idempotent: a second call does not duplicate the import or the Register
// invocation.
func AddImportAndRegister(src []byte, importPath, pkgName string) ([]byte, error) {
	if strings.TrimSpace(importPath) == "" || !isGoIdent(pkgName) {
		return nil, fmt.Errorf("resourcegen: invalid import %q package %q", importPath, pkgName)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, serverMainRel, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("resourcegen: parse %s: %w", serverMainRel, err)
	}
	addImport(file, importPath)
	if err := insertRegisterCall(file, pkgName); err != nil {
		return nil, err
	}
	return formatFile(fset, file)
}

// AddAutoMigrateModel adds `&<pkg>.<type>{}` to the AutoMigrate call in
// internal/platform and ensures importPath is imported. Idempotent.
func AddAutoMigrateModel(src []byte, importPath, pkgName, typeName string) ([]byte, error) {
	if strings.TrimSpace(importPath) == "" || !isGoIdent(pkgName) || !isExportedIdent(typeName) {
		return nil, fmt.Errorf("resourcegen: invalid AutoMigrate target %s.%s", pkgName, typeName)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, platformDBRel, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("resourcegen: parse %s: %w", platformDBRel, err)
	}
	addImport(file, importPath)
	if err := insertAutoMigrateArg(file, pkgName, typeName); err != nil {
		return nil, err
	}
	return formatFile(fset, file)
}

func formatFile(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("resourcegen: format: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("resourcegen: format source: %w", err)
	}
	return formatted, nil
}

func addImport(file *ast.File, importPath string) {
	quoted := strconv.Quote(importPath)
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == quoted {
			return
		}
	}
	spec := &ast.ImportSpec{
		Path: &ast.BasicLit{Kind: token.STRING, Value: quoted},
	}
	file.Imports = append(file.Imports, spec)

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		gen.Specs = append(gen.Specs, spec)
		if len(gen.Specs) > 1 && gen.Lparen == token.NoPos {
			gen.Lparen = 1
		}
		return
	}

	decl := &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: 1,
		Specs:  []ast.Spec{spec},
	}
	file.Decls = append([]ast.Decl{decl}, file.Decls...)
}

func insertRegisterCall(file *ast.File, pkgName string) error {
	if countSelectorCalls(file, pkgName, "Register") > 0 {
		return nil
	}

	stmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(pkgName),
				Sel: ast.NewIdent("Register"),
			},
			Args: []ast.Expr{ast.NewIdent("app")},
		},
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		insertAt := -1
		afterRegister := -1
		beforeRun := -1
		for i, item := range fn.Body.List {
			if isSelectorCallStmt(item, "", "Register") {
				afterRegister = i + 1
			}
			if isSelectorCallStmt(item, "framework", "Run") {
				beforeRun = i
			}
		}
		switch {
		case afterRegister >= 0:
			insertAt = afterRegister
		case beforeRun >= 0:
			insertAt = beforeRun
		}
		if insertAt < 0 {
			continue
		}
		list := fn.Body.List
		fn.Body.List = append(list[:insertAt:insertAt], append([]ast.Stmt{stmt}, list[insertAt:]...)...)
		return nil
	}
	return fmt.Errorf("resourcegen: no product.Register(app) or framework.Run(app) registration point in %s", serverMainRel)
}

func insertAutoMigrateArg(file *ast.File, pkgName, typeName string) error {
	found := false
	var visitErr error
	ast.Inspect(file, func(n ast.Node) bool {
		if visitErr != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "AutoMigrate" {
			return true
		}
		found = true
		if hasAutoMigrateModel(call, pkgName, typeName) {
			return false
		}
		call.Args = append(call.Args, &ast.UnaryExpr{
			Op: token.AND,
			X: &ast.CompositeLit{
				Type: &ast.SelectorExpr{
					X:   ast.NewIdent(pkgName),
					Sel: ast.NewIdent(typeName),
				},
			},
		})
		return false
	})
	if visitErr != nil {
		return visitErr
	}
	if !found {
		return fmt.Errorf("resourcegen: no AutoMigrate call in %s", platformDBRel)
	}
	return nil
}

func hasAutoMigrateModel(call *ast.CallExpr, pkgName, typeName string) bool {
	for _, arg := range call.Args {
		unary, ok := arg.(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			continue
		}
		lit, ok := unary.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == pkgName && sel.Sel.Name == typeName {
			return true
		}
	}
	return false
}

func countSelectorCalls(file *ast.File, pkgName, selName string) int {
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isSelectorCall(call, pkgName, selName) {
			count++
		}
		return true
	})
	return count
}

func isSelectorCallStmt(stmt ast.Stmt, pkgName, selName string) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isSelectorCall(call, pkgName, selName)
}

func isSelectorCall(call *ast.CallExpr, pkgName, selName string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != selName {
		return false
	}
	if pkgName == "" {
		return true
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == pkgName
}

// CountRegisterCalls reports how many `<pkg>.Register(...)` calls appear in src.
func CountRegisterCalls(src []byte, pkgName string) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		return 0, err
	}
	return countSelectorCalls(file, pkgName, "Register"), nil
}
