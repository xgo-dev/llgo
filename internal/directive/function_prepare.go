/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package directive

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/ssa/abi"
)

// PreparePackage parses function attributes and shared symbol annotations.
// It owns declaration/name association and publishes only after validation.
// The returned values feed the existing linker indexes, including variables.
func (p *Functions) PreparePackage(fset *token.FileSet, pkg *types.Package, files []*ast.File, cPackage, exportRename bool) (map[string]FunctionDirectives, error) {
	path := abi.PathOf(pkg)
	aliases := receiverAliases(files)
	declarations := make(map[*ast.FuncDecl]*FunctionDirectives)
	names := make(map[string]*FunctionDirectives)
	bindings := make(map[string]*FunctionDirectives)
	syms := make(map[string]string)
	linkDocs := make(map[*ast.CommentGroup]bool)
	for _, file := range files {
		for _, node := range file.Decls {
			switch decl := node.(type) {
			case *ast.FuncDecl:
				fullName, inPkgName := astFuncName(path, decl, aliases)
				syms[inPkgName] = fullName
				if alias := ParenthesizedMethodName(inPkgName); alias != "" {
					syms[alias] = fullName
				}
				if decl.Doc != nil {
					linkDocs[decl.Doc] = true
				}
				info, err := parseFunction(fset, decl.Doc, inPkgName, exportRename)
				if err != nil {
					return nil, err
				}
				if info.Linkname == "" && cPackage && decl.Recv == nil && token.IsExported(inPkgName) {
					info.Linkname = strings.TrimPrefix(inPkgName, "X")
					info.ExportName = info.Linkname
				}
				registerFunction(declarations, names, decl, fullName, info)
				bindings[fullName] = names[fullName]
			case *ast.GenDecl:
				if decl.Tok != token.VAR {
					continue
				}
				for _, spec := range decl.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						fullName := path + "." + name.Name
						syms[name.Name] = fullName
						bindings[fullName] = new(FunctionDirectives)
					}
				}
				if len(decl.Specs) == 1 {
					vars := decl.Specs[0].(*ast.ValueSpec).Names
					if len(vars) == 1 {
						if decl.Doc != nil {
							linkDocs[decl.Doc] = true
						}
						link, export, err := parseBinding(fset, decl.Doc, vars[0].Name, exportRename)
						if err != nil {
							return nil, err
						}
						bindings[path+"."+vars[0].Name] = &FunctionDirectives{Linkname: link, ExportName: export}
					}
				}
			}
		}
	}
	if err := collectPackageLinknames(fset, files, syms, linkDocs, bindings); err != nil {
		return nil, err
	}
	result := make(map[string]FunctionDirectives, len(bindings))
	for name, info := range bindings {
		result[name] = *info
	}
	p.sources.set(pkg, declarations, names)
	return result, nil
}

// Register publishes an already prepared declaration, including empty attributes
// for generated AST helpers. It must run before any wrapper for decl is used.
func (p *Functions) Register(pkg *types.Package, decl *ast.FuncDecl, name string, info FunctionDirectives) {
	p.sources.mu.Lock()
	defer p.sources.mu.Unlock()
	names := p.sources.functionNames[pkg]
	if names == nil {
		names = make(map[string]*FunctionDirectives)
		p.sources.functionNames[pkg] = names
	}
	registerFunction(p.sources.functions, names, decl, name, info)
}

func registerFunction(declarations map[*ast.FuncDecl]*FunctionDirectives, names map[string]*FunctionDirectives, decl *ast.FuncDecl, name string, info FunctionDirectives) {
	declarations[decl] = &info
	names[name] = &info
}

// HasExternalBody conservatively rejects source bodies with foreign bindings
// (or missing preparation). Build's runtime-free proof uses source-body facts,
// not a patch's effective call binding; SSA consumers use Function instead.
func (p *Functions) HasExternalBody(files []*ast.File) bool {
	for _, file := range files {
		for _, node := range file.Decls {
			if fn, ok := node.(*ast.FuncDecl); ok && fn.Body != nil {
				info, prepared := p.sources.declaration(fn)
				if !prepared || info.Linkname != "" && !strings.HasPrefix(info.Linkname, "llgo.") {
					return true
				}
			}
		}
	}
	return false
}

func recvTypeName(typ ast.Expr) string {
retry:
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return trecvTypeName(t.X, t.Index)
	case *ast.IndexListExpr:
		return trecvTypeName(t.X, t.Indices...)
	case *ast.ParenExpr:
		typ = t.X
		goto retry
	}
	panic("unreachable")
}

// TODO(xsw): support generic type
func trecvTypeName(t ast.Expr, indices ...ast.Expr) string {
	_ = indices
	return t.(*ast.Ident).Name
}

// inPkgName:
// - func: name
// - method: T.name, (*T).name
// fullName:
// - func: pkg.name
// - method: pkg.(T).name, pkg.(*T).name
func astFuncName(pkgPath string, fn *ast.FuncDecl, aliases map[string]ast.Expr) (fullName, inPkgName string) {
	name := fn.Name.Name
	if recv := fn.Recv; recv != nil && len(recv.List) == 1 {
		t := ast.Unparen(recv.List[0].Type)
		pointer := false
		if tp, ok := t.(*ast.StarExpr); ok {
			t, pointer = tp.X, true
		}
		recvName := recvTypeName(t)
		canonical := recvName
		canonicalPtr := pointer
		// Preloading runs before type checking. Resolve local receiver aliases
		// from syntax, including forward and cross-file references. Bound the
		// walk so invalid alias cycles are left for go/types to diagnose.
		for n := 0; n < len(aliases); n++ {
			rhs, ok := aliases[canonical]
			if !ok {
				break
			}
			rhs = ast.Unparen(rhs)
			ptr, isPtr := rhs.(*ast.StarExpr)
			if isPtr {
				rhs = ast.Unparen(ptr.X)
			}
			ident, ok := rhs.(*ast.Ident)
			if !ok {
				break // Stop local alias resolution; go/types handles this type.
			}
			canonical, canonicalPtr = ident.Name, canonicalPtr || isPtr
		}
		if pointer {
			recvName = "(*" + recvName + ")"
		}
		if canonicalPtr {
			canonical = "(*" + canonical + ")"
		}
		return pkgPath + "." + canonical + "." + name, recvName + "." + name
	}
	return pkgPath + "." + name, name
}

func receiverAliases(files []*ast.File) map[string]ast.Expr {
	aliases := make(map[string]ast.Expr)
	for _, file := range files {
		for _, decl := range file.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.TYPE {
				for _, spec := range decl.Specs {
					if spec := spec.(*ast.TypeSpec); spec.Assign.IsValid() {
						aliases[spec.Name.Name] = spec.Type
					}
				}
			}
		}
	}
	return aliases
}

// collectPackageLinknames follows cmd/compile's package-scoped go:linkname
// behavior and diagnoses method names that would otherwise be silently ignored.
func collectPackageLinknames(fset *token.FileSet, files []*ast.File, syms map[string]string, linkDocs map[*ast.CommentGroup]bool, bindings map[string]*FunctionDirectives) error {
	for _, file := range files {
		importsUnsafe := false
		for _, imp := range file.Imports {
			if imp.Path.Value == `"unsafe"` {
				importsUnsafe = true
				break
			}
		}
		for _, group := range file.Comments {
			for _, item := range ParseGroup(group) {
				if item.Name != "llgo:link" && (item.Name != "go:linkname" || !importsUnsafe) {
					continue
				}
				fields := strings.Fields(item.Args)
				if item.Name == "llgo:link" {
					// LLGo links must be attached to the declaration they name.
					// Declaration scanning above has already validated and stored them.
					if len(fields) < 2 {
						return fmt.Errorf("%s: //llgo:link requires a local name and a target", fset.Position(item.Pos))
					}
					if linkDocs[group] {
						continue
					}
					return fmt.Errorf("%s: //llgo:link local name %q is not attached to a declaration", fset.Position(item.Pos), fields[0])
				}
				if len(fields) < 2 {
					continue // Single-name go:linkname is a valid export annotation.
				}
				if fullName, ok := syms[fields[0]]; ok {
					if info := bindings[fullName]; info != nil {
						info.Linkname = strings.Join(fields[1:], " ")
					}
				} else if strings.Contains(fields[0], ".") {
					return fmt.Errorf("%s: //go:linkname local method %q not found", fset.Position(item.Pos), fields[0])
				}
			}
		}
	}
	return nil
}
