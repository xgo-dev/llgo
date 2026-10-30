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
)

// FunctionDirectives is the prepared value attached to a source function.
// Empty strings mean no binding; valid bindings always have a nonempty name.
// No AST, object, package, or derived symbol identity is duplicated here.
type FunctionDirectives struct {
	NoInline       bool
	NoSplit        bool
	UintptrEscapes bool
	ClosureEnv     bool
	NoInterface    bool
	WasmModule     string
	WasmName       string
	Linkname       string
	ExportName     string
}

// parseFunction keeps the existing per-directive attachment rules.
// It returns a value, with no discovery cache or declaration lookup machinery.
func parseFunction(fset *token.FileSet, doc *ast.CommentGroup, inPkgName string, exportRename bool) (info FunctionDirectives, err error) {
	info.Linkname, info.ExportName, err = parseBinding(fset, doc, inPkgName, exportRename)
	if err != nil {
		return info, err
	}
	directives := ParseGroup(doc)
	wasmImportSeen := false
	for n := len(directives) - 1; n >= 0; n-- {
		item := directives[n]
		switch item.Name {
		case "go:noinline":
			info.NoInline = true
		case "go:nosplit":
			info.NoSplit = true
		case "go:uintptrescapes":
			info.UintptrEscapes = true
		case "llgo:env":
			info.ClosureEnv = true
		case "go:wasmimport":
			if wasmImportSeen {
				continue
			}
			wasmImportSeen = true
			if fields := strings.Fields(item.Args); len(fields) == 2 {
				info.WasmModule, info.WasmName = fields[0], fields[1]
			}
		}
	}
	if doc != nil {
		for n := len(doc.List) - 1; n >= 0; n-- {
			line := doc.List[n].Text
			if line == "//go:nointerface" {
				info.NoInterface = true
				break
			}
			if !strings.HasPrefix(line, "//go:") {
				break
			}
		}
	}
	return info, nil
}

// A value receiver can be written either T.M or (T).M in a link directive.
// This also applies when T is an alias for a pointer type.
func ParenthesizedMethodName(name string) string {
	if i := strings.IndexByte(name, '.'); i > 0 && name[0] != '(' {
		return "(" + name[:i] + ")" + name[i:]
	}
	return ""
}

// Keep the source receiver spelling for matching directives, but register them
// under the unaliased receiver used by SSA and the linker.
func receiverName(typ types.Type) (canonical, source string) {
	switch t := typ.(type) {
	case *types.Alias:
		canonical, _ = receiverName(types.Unalias(t))
		return canonical, t.Obj().Name()
	case *types.Pointer:
		canonical, source = receiverName(t.Elem())
		return "(*" + canonical + ")", "(*" + source + ")"
	case *types.Named:
		return t.Obj().Name(), t.Obj().Name()
	}
	panic(fmt.Errorf("invalid recv type: %v", typ))
}

// FuncName returns the canonical full name and source receiver spelling.
func FuncName(pkgPath string, fn *types.Func) (fullName, inPkgName string) {
	sig := fn.Type().(*types.Signature)
	name := fn.Name()
	if recv := sig.Recv(); recv != nil {
		canonical, source := receiverName(recv.Type())
		return pkgPath + "." + canonical + "." + name, source + "." + name
	}
	return pkgPath + "." + name, name
}

// parseBinding is shared by function and variable preparation; function-only
// pragmas are interpreted exclusively by parseFunction.
func parseBinding(fset *token.FileSet, doc *ast.CommentGroup, inPkgName string, exportRename bool) (linkname, exportName string, err error) {
	directives := ParseGroup(doc)
	for n := len(directives) - 1; n >= 0; n-- {
		item := directives[n]
		switch item.Name {
		case "go:linkname", "llgo:link":
			fields := strings.Fields(item.Args)
			if item.Name == "llgo:link" {
				if len(fields) < 2 {
					return "", "", fmt.Errorf("%s: //llgo:link requires a local name and a target", fset.Position(item.Pos))
				}
				if fields[0] != inPkgName && fields[0] != ParenthesizedMethodName(inPkgName) {
					return "", "", fmt.Errorf("%s: //llgo:link local name %q does not match declaration %q", fset.Position(item.Pos), fields[0], inPkgName)
				}
			}
			if linkname == "" && len(fields) >= 2 && (fields[0] == inPkgName || fields[0] == ParenthesizedMethodName(inPkgName)) {
				linkname = strings.Join(fields[1:], " ")
			}
		case "export":
			if linkname != "" || item.Args == "" {
				continue
			}
			if item.Args != inPkgName && !exportRename {
				return "", "", fmt.Errorf("export comment has wrong name %q", item.Args)
			}
			linkname, exportName = item.Args, item.Args

		}
	}
	return linkname, exportName, nil
}
