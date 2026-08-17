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
// none exists, immediately before `framework.Run(app)` — including the
// `if err := framework.Run(app); err != nil` form used by the app template.
// The edit is idempotent: a second call does not duplicate the import or the
// Register invocation.
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
		if insertIntoBlock(fn.Body, stmt) {
			return nil
		}
	}
	return fmt.Errorf("resourcegen: no product.Register(app) or framework.Run(app) registration point in %s", serverMainRel)
}

func insertIntoBlock(block *ast.BlockStmt, stmt ast.Stmt) bool {
	if block == nil {
		return false
	}
	afterRegister := -1
	beforeRun := -1
	for i, item := range block.List {
		if stmtAnchorsSelectorCall(item, "", "Register") {
			afterRegister = i + 1
		}
		if stmtAnchorsSelectorCall(item, "framework", "Run") {
			beforeRun = i
		}
	}
	insertAt := -1
	switch {
	case afterRegister >= 0:
		insertAt = afterRegister
	case beforeRun >= 0:
		insertAt = beforeRun
	}
	if insertAt >= 0 {
		list := block.List
		block.List = append(list[:insertAt:insertAt], append([]ast.Stmt{stmt}, list[insertAt:]...)...)
		return true
	}
	for _, item := range block.List {
		for _, child := range nestedBlocks(item) {
			if insertIntoBlock(child, stmt) {
				return true
			}
		}
	}
	return false
}

func nestedBlocks(stmt ast.Stmt) []*ast.BlockStmt {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return []*ast.BlockStmt{s}
	case *ast.IfStmt:
		var blocks []*ast.BlockStmt
		if s.Body != nil {
			blocks = append(blocks, s.Body)
		}
		if s.Else != nil {
			blocks = append(blocks, nestedBlocks(s.Else)...)
		}
		return blocks
	case *ast.ForStmt:
		if s.Body != nil {
			return []*ast.BlockStmt{s.Body}
		}
	case *ast.RangeStmt:
		if s.Body != nil {
			return []*ast.BlockStmt{s.Body}
		}
	}
	return nil
}

// stmtAnchorsSelectorCall reports whether stmt itself wraps a selector call,
// including IfStmt.Init and AssignStmt RHS. Nested bodies are ignored so the
// caller can recurse and insert at the inner block.
func stmtAnchorsSelectorCall(stmt ast.Stmt, pkgName, selName string) bool {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return exprContainsSelectorCall(s.X, pkgName, selName)
	case *ast.AssignStmt:
		for _, rhs := range s.Rhs {
			if exprContainsSelectorCall(rhs, pkgName, selName) {
				return true
			}
		}
	case *ast.IfStmt:
		if s.Init != nil && stmtAnchorsSelectorCall(s.Init, pkgName, selName) {
			return true
		}
		if exprContainsSelectorCall(s.Cond, pkgName, selName) {
			return true
		}
	case *ast.ReturnStmt:
		for _, result := range s.Results {
			if exprContainsSelectorCall(result, pkgName, selName) {
				return true
			}
		}
	}
	return false
}

func exprContainsSelectorCall(expr ast.Expr, pkgName, selName string) bool {
	if expr == nil {
		return false
	}
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isSelectorCall(call, pkgName, selName) {
			found = true
			return false
		}
		return true
	})
	return found
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
