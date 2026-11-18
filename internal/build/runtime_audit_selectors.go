// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package build

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
)

func runtimeAuditLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		name, _ := strconv.Unquote(lit.Value)
		return name
	}
	return ""
}

func runtimeAuditCandidates(file *ast.File, expr ast.Expr) ([]string, bool) {
	if name := runtimeAuditLiteral(expr); name != "" {
		return []string{name}, false
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() || len(call.Args) < 2 {
		return nil, false
	}
	checked := false
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		checked = file.Name.Name == "ssa" && fun.Name == "RuntimeSymbol"
		if checked && fun.Obj != nil {
			// ParseFile resolves a same-file package function, but leaves
			// cross-file declarations unresolved. Both denote the helper;
			// local bindings and package variables must still be rejected.
			checked = fun.Obj.Kind == ast.Fun && file.Scope != nil && file.Scope.Lookup(fun.Name) == fun.Obj
		}
	case *ast.SelectorExpr:
		if recv, ok := fun.X.(*ast.Ident); ok && fun.Sel.Name == "RuntimeSymbol" && recv.Obj == nil {
			for _, imp := range file.Imports {
				if imp.Path.Value != `"github.com/xgo-dev/llgo/ssa"` {
					continue
				}
				name := "ssa"
				if imp.Name != nil {
					name = imp.Name.Name
				}
				checked = recv.Name == name
			}
		}
	}
	if !checked {
		return nil, false
	}
	var names []string
	for _, arg := range call.Args[1:] {
		name := runtimeAuditLiteral(arg)
		if name == "" || slices.Contains(names, name) {
			return nil, false
		}
		names = append(names, name)
	}
	return names, true
}

// These lookup APIs forward the name unchanged to another audited API. Match
// the complete one-return body and parameter, rather than exempting a function
// by name: extra statements or computed arguments must be audited normally.
func runtimeAuditForwarders(pkg, filename string, file *ast.File) map[*ast.CallExpr]string {
	out := make(map[*ast.CallExpr]string)
	if pkg != "ssa" || filename != "package.go" {
		return out
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || (fn.Name.Name != "rtType" && fn.Name.Name != "RuntimeFunc") ||
			fn.Body == nil || len(fn.Body.List) != 1 || len(fn.Type.Params.List) != 1 ||
			len(fn.Type.Params.List[0].Names) != 1 || fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
			continue
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		call, ok := ret.Results[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		receiver := fn.Recv.List[0].Names[0].Name
		member := func(call *ast.CallExpr, name string) bool {
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
				return false
			}
			ident, ok := sel.X.(*ast.Ident)
			return ok && ident.Name == receiver && sel.Sel.Name == name
		}
		callee := "rtFunc"
		if fn.Name.Name == "rtType" {
			if !member(call, "rawType") {
				continue
			}
			call, ok = call.Args[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			callee = "rtNamed"
		}
		if !member(call, callee) {
			continue
		}
		if arg, ok := call.Args[0].(*ast.Ident); ok && arg.Name == fn.Type.Params.List[0].Names[0].Name {
			out[call] = fn.Name.Name + " -> " + callee
		}
	}
	return out
}
