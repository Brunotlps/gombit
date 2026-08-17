package commandgen

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

const registerCommandsName = "RegisterCommands"

// AddImportAndRegister adds importPath and `<pkg>.RegisterCommands(root)` next
// to an existing `*.RegisterCommands(root)` call, or immediately after
// `cli.NewRoot`. Idempotent. `cli.Execute` is not a registration point.
func AddImportAndRegister(src []byte, importPath, pkgName string) ([]byte, error) {
	if strings.TrimSpace(importPath) == "" || !isGoIdent(pkgName) {
		return nil, fmt.Errorf("commandgen: invalid import %q package %q", importPath, pkgName)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, gombitMainRel, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("commandgen: parse %s: %w", gombitMainRel, err)
	}
	addImport(file, importPath)
	if err := insertRegisterCall(file, pkgName); err != nil {
		return nil, err
	}
	return formatFile(fset, file)
}

// AddCommandCall appends `cli.AddCommand(root, <constructor>())` inside
// `func RegisterCommands` when it is not already present. Idempotent.
func AddCommandCall(src []byte, constructor string) ([]byte, error) {
	if !isExportedIdent(constructor) {
		return nil, fmt.Errorf("commandgen: invalid constructor %q", constructor)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, commandsFileName, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("commandgen: parse %s: %w", commandsFileName, err)
	}
	addImport(file, "github.com/LAA-Software-Engineering/gombit/cli")
	if err := insertAddCommandCall(file, constructor); err != nil {
		return nil, err
	}
	return formatFile(fset, file)
}

func formatFile(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("commandgen: format: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("commandgen: format source: %w", err)
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
	if countSelectorCalls(file, pkgName, registerCommandsName) > 0 {
		return nil
	}

	stmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(pkgName),
				Sel: ast.NewIdent(registerCommandsName),
			},
			Args: []ast.Expr{ast.NewIdent("root")},
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
	return fmt.Errorf("commandgen: no product.RegisterCommands(root) or cli.NewRoot registration point in %s", gombitMainRel)
}

func insertIntoBlock(block *ast.BlockStmt, stmt ast.Stmt) bool {
	if block == nil {
		return false
	}
	afterRegister := -1
	afterNewRoot := -1
	for i, item := range block.List {
		if stmtAnchorsSelectorCall(item, "", registerCommandsName) {
			afterRegister = i + 1
		}
		if stmtAnchorsSelectorCall(item, "cli", "NewRoot") {
			afterNewRoot = i + 1
		}
	}
	insertAt := -1
	switch {
	case afterRegister >= 0:
		insertAt = afterRegister
	case afterNewRoot >= 0:
		insertAt = afterNewRoot
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

func insertAddCommandCall(file *ast.File, constructor string) error {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != registerCommandsName || fn.Body == nil {
			continue
		}
		if hasConstructorCall(fn.Body, constructor) {
			return nil
		}
		fn.Body.List = append(fn.Body.List, addCommandStmt(constructor))
		return nil
	}
	return fmt.Errorf("commandgen: no RegisterCommands function in %s", commandsFileName)
}

func addCommandStmt(constructor string) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("cli"),
				Sel: ast.NewIdent("AddCommand"),
			},
			Args: []ast.Expr{
				ast.NewIdent("root"),
				&ast.CallExpr{Fun: ast.NewIdent(constructor)},
			},
		},
	}
}

func hasConstructorCall(block *ast.BlockStmt, constructor string) bool {
	if block == nil {
		return false
	}
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if ok && ident.Name == constructor {
			found = true
			return false
		}
		return true
	})
	return found
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

// CountRegisterCalls reports how many `<pkg>.RegisterCommands(...)` calls appear in src.
func CountRegisterCalls(src []byte, pkgName string) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		return 0, err
	}
	return countSelectorCalls(file, pkgName, registerCommandsName), nil
}

// CountConstructorCalls reports how many `Constructor()` calls appear in src.
func CountConstructorCalls(src []byte, constructor string) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, commandsFileName, src, 0)
	if err != nil {
		return 0, err
	}
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if ok && ident.Name == constructor {
			count++
		}
		return true
	})
	return count, nil
}
