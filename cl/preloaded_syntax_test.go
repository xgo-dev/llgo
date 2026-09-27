//go:build !llgo

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
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goembed"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func TestPreloadedSyntaxFeedsBackendWithoutLateDiscovery(t *testing.T) {
	const source = `package C

//export callback
func Callback() {}

func XDefault() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "preloaded.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := newLocalityTypeInfo()
	imp := importer.Default()
	pkg, err := (&types.Config{Importer: imp}).Check("example.com/C", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	coordinator := ssatest.NewProgramEx(t, nil, imp)
	if err := ParsePkgSyntaxWithOptions(coordinator, fset, pkg, files, Options{ExportRename: true}); err != nil {
		t.Fatal(err)
	}
	backend := coordinator.NewBackendProgram()
	defer backend.Dispose()

	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()
	compiled, _, err := NewPackageExWithEmbedMetaOptions(
		backend, nil, nil, nil, ssaPkg, files, goembed.VarMap{}, false,
		Options{ExportRename: true, CExportWrappers: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	for fullName, want := range map[string]string{
		"example.com/C.Callback": "callback",
		"example.com/C.XDefault": "Default",
	} {
		if link, ok := backend.Linkname(fullName); !ok || link != want {
			t.Errorf("Linkname(%q) = (%q, %v), want (%q, true)", fullName, link, ok, want)
		}
		if export, ok := compiled.ExportFuncs()[fullName]; !ok || export != want {
			t.Errorf("ExportFuncs()[%q] = (%q, %v), want (%q, true)", fullName, export, ok, want)
		}
		if fn := compiled.FuncOf(fullName); fn == nil {
			t.Errorf("FuncOf(%q) = nil, want wrapped implementation", fullName)
		}
		if fn := compiled.FuncOf(want); fn != nil {
			t.Errorf("FuncOf(%q) = %q, want final-link wrapper only", want, fn.Name())
		}
	}
}

func TestCompilationRequiresPreparedSyntax(t *testing.T) {
	pkg, _, files := buildGoSSAPkg(t, "package p\nfunc F() {}\n")
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	_, err := NewPackage(prog, pkg, files)
	if err == nil || !strings.Contains(err.Error(), "prepare directive records before compilation") {
		t.Fatalf("unprepared compilation error = %v", err)
	}
	if prog.PackageDirectives(pkg.Pkg) != nil {
		t.Fatal("compilation discovered source records")
	}
}
