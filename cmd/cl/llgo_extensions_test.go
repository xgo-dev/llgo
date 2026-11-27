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
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen/ssatest"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
)

func TestNativeSSAReceiverNilChecksWithoutSideTable(t *testing.T) {
	const src = `package foo
type T struct { x int }
func (*T) M(int) {}
type U struct { pad int; T }
func arg() int { return 1 }
func deref(p *T) { (*p).M(arg()) }
func promoted(p *U) { p.M(arg()) }
func bound(p *T) func(int) { return (*p).M }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "receiver.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("foo", "foo"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	// Deliberately pass empty Options{} (no ReceiverNilChecks side table).
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{})
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"deref", "promoted", "bound"} {
		ir := mustNamedFunction(t, mod, "foo."+name).String()
		if !strings.Contains(ir, "AssertNilDeref") {
			t.Fatalf("%s lost native ssa.RecvNilCheck guard:\n%s", name, ir)
		}
	}
}

func TestNativeSSAStaticByteArrayWithoutASTStrip(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nvar Table = [%d]byte{0: 11, %s %d: 22}\n\nfunc Use() byte { return Table[0] + Table[%d] }\n",
		n, strings.Repeat("3, ", n-2), n-1, n-1)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("p", "p"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	// Verify AST was never stripped and Global.StaticBytes was populated natively.
	vs := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(vs.Values) != 1 {
		t.Fatalf("AST was unexpectedly mutated: Values=%d", len(vs.Values))
	}
	if g := ssaPkg.Var("Table"); len(g.StaticBytes) != n {
		t.Fatalf("Table.StaticBytes len = %d, want %d", len(g.StaticBytes), n)
	}
	llprog := ssatest.NewProgramEx(t, nil, nil)
	llprog.TypeSizes(types.SizesFor("gc", "arm64"))
	// Deliberately pass empty Options{} (no StaticByteArrays side table).
	ret, _, err := NewPackageExWithEmbedMetaOptions(llprog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ir := ret.String()
	if !strings.Contains(ir, fmt.Sprintf("@p.Table = global [%d x i8] c\"", n)) {
		t.Fatalf("expected constant byte array initializer from ssa.Global.StaticBytes:\n%s", ir)
	}
	if strings.Count(ir, "store i8") > 16 {
		t.Fatalf("init still emitted elementwise stores:\n%s", ir)
	}
}

func TestImportedClosureEnvPreservedViaExportData(t *testing.T) {
	const depSrc = `package dep

//llgo:env
func EnvTarget(x int) int { return x }
`
	const rootSrc = `package root

import "example.com/dep"

func Call(x int) int { return dep.EnvTarget(x) }
func CallViaValue(x int) int { f := dep.EnvTarget; return f(x) }
`
	fset := token.NewFileSet()
	depFile, err := parser.ParseFile(fset, "dep.go", depSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	depPkg, err := (&types.Config{}).Check("example.com/dep", fset, []*ast.File{depFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var exported bytes.Buffer
	if err := exportdata.Write(&exported, fset, depPkg); err != nil {
		t.Fatalf("exportdata.Write: %v", err)
	}
	importedDep, err := exportdata.Read(bytes.NewReader(exported.Bytes()), fset, map[string]*types.Package{}, depPkg.Path())
	if err != nil {
		t.Fatalf("exportdata.Read: %v", err)
	}

	rootFile, err := parser.ParseFile(fset, "root.go", rootSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	imp := importerFunc(func(path string) (*types.Package, error) {
		if path == importedDep.Path() {
			return importedDep, nil
		}
		return nil, types.Error{Msg: "unexpected import " + path}
	})
	rootSSA, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, types.NewPackage("example.com/root", "root"), []*ast.File{rootFile}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	ret, _, err := newPackageEx(prog, nil, nil, nil, rootSSA, []*ast.File{rootFile}, nil, false, Options{})
	if err != nil {
		t.Fatal(err)
	}
	mod := ret.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	decl := mustNamedFunction(t, mod, "example.com/dep.EnvTarget")
	if got := len(decl.Params()); got != 2 {
		t.Fatalf("imported //llgo:env declaration param count = %d, want 2 (env + x):\n%s", got, decl.String())
	}
}
