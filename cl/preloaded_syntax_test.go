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

package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goembed"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func TestPreparedSyntaxFeedsCompilationEntryPoints(t *testing.T) {
	for _, pkgName := range []string{"C", "callbacks"} {
		for _, entry := range []string{"one-shot", "prepared-one-shot", "backend", "patched-backend"} {
			t.Run(pkgName+"/"+entry, func(t *testing.T) {
				source := `package ` + pkgName + `

//export callback
func Callback() {}

func XDefault() {}

func Plain() {}

//export counter
var Counter int32

//go:linkname Value C.shared_value
var Value int32
`
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "preloaded.go", source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				files := []*ast.File{file}
				info := newLocalityTypeInfo()
				imp := importer.Default()
				pkg, err := (&types.Config{Importer: imp}).Check("example.com/"+pkgName, fset, files, info)
				if err != nil {
					t.Fatal(err)
				}
				goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
				ssaPkg := goProg.CreatePackage(pkg, files, info, true)
				ssaPkg.Build()
				coordinator := ssatest.NewProgramEx(t, nil, imp)
				preparedTypes := pkg
				var patches Patches
				if entry == "patched-backend" {
					// The effective patch package has its own preparation identity.
					// An empty original snapshot must not hide its exports.
					coordinator.MarkPackageSyntaxParsed(pkg, nil)
					preparedTypes = types.NewPackage(pkg.Path(), pkg.Name())
					alt := goProg.CreatePackage(types.NewPackage(pkg.Path()+"/alt", pkg.Name()), nil, nil, true)
					alt.Build()
					patches = Patches{pkg.Path(): {Alt: alt, Types: preparedTypes}}
				}
				if entry != "one-shot" {
					if err := ParsePkgSyntaxWithOptions(coordinator, fset, preparedTypes, files, Options{ExportRename: true}); err != nil {
						t.Fatal(err)
					}
					// Once prepared, declaration bindings must not be rediscovered by
					// either compilation entrypoint, even with PreloadedSyntax=false.
					for _, decl := range file.Decls {
						switch decl := decl.(type) {
						case *ast.FuncDecl:
							decl.Doc = nil
						case *ast.GenDecl:
							decl.Doc = nil
						}
					}
					file.Comments = nil
					// Export preservation must not need function declarations either.
					decls := make([]ast.Decl, 0, len(file.Decls))
					for _, decl := range file.Decls {
						if _, function := decl.(*ast.FuncDecl); !function {
							decls = append(decls, decl)
						}
					}
					file.Decls = decls
				}
				backend := coordinator
				preloaded := entry == "backend" || entry == "patched-backend"
				if preloaded {
					backend = coordinator.NewBackendProgram()
					defer backend.Dispose()
				}

				compiled, _, err := NewPackageExWithEmbedMetaOptions(
					backend, nil, patches, nil, ssaPkg, files, goembed.VarMap{}, false,
					Options{ExportRename: true, CExportWrappers: true, PreloadedSyntax: preloaded},
				)
				if err != nil {
					t.Fatal(err)
				}
				exports := map[string]string{"Callback": "callback"}
				if pkgName == "C" {
					exports["XDefault"], exports["Plain"] = "Default", "Plain"
				}
				for name, cName := range exports {
					fullName := pkg.Path() + "." + name
					if link, ok := backend.Linkname(fullName); !ok || link != cName {
						t.Errorf("Linkname(%q) = (%q, %v), want (%q, true)", fullName, link, ok, cName)
					}
					if export, ok := compiled.LookupExport(fullName); !ok || export != cName {
						t.Errorf("LookupExport(%q) = (%q, %v), want (%q, true)", fullName, export, ok, cName)
					}
					implName, absentName := fullName, cName
					if pkgName == "C" {
						implName, absentName = cName, fullName
					}
					if fn := compiled.FuncOf(implName); fn == nil || !fn.HasBody() {
						t.Errorf("FuncOf(%q) has no implementation", implName)
					}
					if fn := compiled.FuncOf(absentName); fn != nil {
						t.Errorf("FuncOf(%q) = %q, want no alternate entry", absentName, fn.Name())
					}
				}
				if pkgName == "C" && !strings.Contains(compiled.String(), "@llvm.compiler.used") {
					t.Fatal("package C lost its direct export reachability roots")
				}
				if got, ok := compiled.LookupExport(pkg.Path() + ".Counter"); !ok || got != "counter" {
					t.Errorf("variable export = %q, want counter", got)
				}
				if got, ok := backend.Linkname(pkg.Path() + ".Value"); !ok || got != "C.shared_value" {
					t.Errorf("variable linkname = (%q, %v), want (C.shared_value, true)", got, ok)
				}
				if got := maps.Collect(compiled.Exports()); len(got) != len(exports)+1 {
					t.Fatalf("unexpected automatic exports for package %s: %v", pkgName, got)
				}
			})
		}
	}
}
