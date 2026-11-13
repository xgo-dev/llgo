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

package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestUnsafeOffsetOwners(t *testing.T) {
	t.Run("package-load-and-rewrite", checkUnsafeOffsetPackageLoad)
	const declarations = `
type GoValue struct { B byte; U uint64 }
//llgo:cdecl
type Native GoValue
type Alias = Native
type Callbacks struct { Head byte; F func(); Middle byte; G func(); Tail uintptr }
//llgo:cdecl
type NativeCallbacks Callbacks
//llgo:cdecl
type CFunc func()
type ExplicitCallbacks struct { Head byte; F CFunc; Middle byte; G CFunc; Tail uintptr }
type Outer struct { Prefix byte; Native }
type Generic[T any] struct { Value T; Tail byte }
type ArrayCallbacks struct { A [2]CFunc; Tail uintptr }
//llgo:cdecl
type NativeArrayCallbacks ArrayCallbacks
type RawInner struct { F func(); Tail byte }
//llgo:cdecl
type NativeOuter struct { Inner RawInner; Tail uintptr }
type Recursive struct { Next *Recursive; Tail uintptr }
//llgo:cdecl
type NativeRecursive Recursive
type ZeroTail struct { N byte; Z [0]uint64 }
type ZeroContainer struct { V ZeroTail; Tail byte }

var goValue GoValue
var native Native
var callbacks Callbacks
var nativeCallbacks NativeCallbacks
`
	llssa.Initialize(llssa.InitAll)
	runtimeTypes, err := importer.For("source", nil).Import(llssa.PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []*llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"}, {GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"}, {GOOS: "linux", GOARCH: "arm"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-w64-windows-gnu"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", LLVMTarget: "wasm32-unknown-emscripten"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", LLVMTarget: "wasm32-unknown-wasip1"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", LLVMTarget: "wasm64-unknown-emscripten"},
	} {
		for _, test := range []struct {
			name, source, errorText string
		}{
			{name: "callback-arrays", source: `const A = unsafe.Offsetof(ArrayCallbacks{}.Tail); const B = unsafe.Offsetof(NativeArrayCallbacks{}.Tail)`},
			{name: "raw-nested", source: `const A = unsafe.Offsetof(NativeOuter{}.Tail)`},
			{name: "recursive", source: `const A = unsafe.Offsetof(NativeRecursive{}.Tail); const B = unsafe.Offsetof(Recursive{}.Tail)`},
			{name: "zero-tail", source: `const A = unsafe.Offsetof(ZeroContainer{}.Tail)`},
			{name: "shared-fields", source: `const A = unsafe.Offsetof(goValue.U); const B = unsafe.Offsetof(native.U)`},
			{name: "shared-fields-reverse", source: `const A = unsafe.Offsetof(native.U); const B = unsafe.Offsetof(goValue.U)`},
			{name: "alias", source: `const A = unsafe.Offsetof(Alias{}.U)`},
			{name: "callbacks", source: `const A = unsafe.Offsetof(callbacks.Tail); const B = unsafe.Offsetof(nativeCallbacks.Tail); const C = unsafe.Offsetof(ExplicitCallbacks{}.Tail)`},
			{name: "callbacks-reverse", source: `const A = unsafe.Offsetof(nativeCallbacks.Tail); const B = unsafe.Offsetof(callbacks.Tail)`},
			{name: "promoted", source: `const A = unsafe.Offsetof(Outer{}.U)`},
			{name: "explicit-nested", source: `const A = unsafe.Offsetof(Outer{}.Native.U)`},
			{name: "nested-query", source: `var items [16]Native; const A = unsafe.Offsetof(items[unsafe.Offsetof(goValue.U)].U)`},
			{name: "array-bound", source: `type Buffer [unsafe.Offsetof(native.U)]byte; const A = unsafe.Offsetof(struct { V Buffer; Tail byte }{}.Tail)`},
			{name: "forward", source: `const A = unsafe.Offsetof(later.U); var later Native`},
			{name: "generic", source: `const A = unsafe.Offsetof(Generic[Native]{}.Tail)`},
			{name: "local", source: `func local() { type Local struct { Value Native; Tail byte }; const A = unsafe.Offsetof(Local{}.Tail) }`},
			{name: "anonymous", source: `const A = unsafe.Offsetof(struct { Value Native; Tail byte }{}.Tail)`},
			{name: "pointer", source: `const A = unsafe.Offsetof((*Native)(nil).U)`},
			{name: "parentheses", source: `const A = (unsafe.Offsetof)(((native.U)))`},
			{name: "repeated-const", source: "const ( A = unsafe.Offsetof(native.U) + iota; B; C )"},
			{name: "repeated-mixed", source: "const ( A = unsafe.Offsetof(native.U) + unsafe.Offsetof(goValue.U) + iota; B; C )"},
			{name: "repeated-promoted", source: "const ( A = unsafe.Offsetof(Outer{}.U) + iota; B; C )"},
			{name: "repeated-nested", source: "var items [16]Native; const ( A = unsafe.Offsetof(items[unsafe.Offsetof(goValue.U)].U) + iota; B; C )"},
			{name: "repeated-generic", source: "const ( A = unsafe.Offsetof(Generic[[iota+1]Native]{}.Tail); B; C )"},
			{name: "ordinary-selector", source: `var read = native.U; const A = unsafe.Offsetof(goValue.U)`},
			{name: "variable-generic", source: `func generic[T any](v Generic[T]) uintptr { return unsafe.Offsetof(v.Tail) }; const A = unsafe.Offsetof(native.U)`},
			{name: "shadowed-builtin", source: `func shadow() { unsafe := struct { Offsetof func(uint64) uintptr }{}; _ = unsafe.Offsetof(native.U) }; const A = unsafe.Offsetof(goValue.U)`},
			{name: "import-alias", source: `const A = mem.Offsetof(native.U)`},
			{name: "dot-import", source: `const A = Offsetof(native.U)`},
			{name: "invalid-field", source: `var bad = unsafe.Offsetof(native.Missing); const A = unsafe.Offsetof(goValue.U)`, errorText: "no single field Missing"},
			{name: "invalid-embedding", source: `type Bad struct { *Native }; var bad = unsafe.Offsetof(Bad{}.U); const A = unsafe.Offsetof(goValue.U)`, errorText: "embedded via a pointer"},
			{name: "invalid-context", source: `var bad string = unsafe.Offsetof(native.U); const A = unsafe.Offsetof(goValue.U)`, errorText: "cannot use"},
		} {
			t.Run(fmt.Sprintf("%s/%s/%s/%s", target.GOOS, target.GOARCH, target.LLVMTarget, test.name), func(t *testing.T) {
				prog := llssa.NewProgram(target)
				defer prog.Dispose()
				fset := token.NewFileSet()
				imports := `import "unsafe"`
				switch test.name {
				case "import-alias":
					imports = `import mem "unsafe"`
				case "dot-import":
					imports = `import . "unsafe"`
				}
				file, err := parser.ParseFile(fset, "offset.go", "package p\n"+imports+declarations+test.source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Uses: make(map[*ast.Ident]types.Object), Defs: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection)}
				pkg := &packages.Package{Types: types.NewPackage("example.com/offset", "p"), TypesInfo: info, Syntax: []*ast.File{file}}
				if err := cl.ParsePkgSyntax(prog, fset, pkg.Types, pkg.Syntax); err != nil {
					t.Fatal(err)
				}
				pkg.TypesSizes = prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
				sizes := withUnsafeOffsetOwner(prog, pkg)
				var errors []error
				conf := &types.Config{Sizes: sizes, Importer: importer.Default(), Error: func(err error) { errors = append(errors, err) }}
				err = types.NewChecker(conf, fset, pkg.Types, info).Files(pkg.Syntax)
				if test.errorText == "" && err != nil {
					t.Fatal(err)
				}
				if test.errorText != "" && (len(errors) != 1 || !strings.Contains(errors[0].Error(), test.errorText)) {
					t.Fatalf("errors=%v, want one containing %q", errors, test.errorText)
				}
				prog.SetRuntime(runtimeTypes)
				checked := 0
				for expr, value := range info.Types {
					call, ok := expr.(*ast.CallExpr)
					if !ok || value.Value == nil || len(call.Args) != 1 {
						continue
					}
					selector, ok := ast.Unparen(call.Args[0]).(*ast.SelectorExpr)
					if !ok {
						continue
					}
					var callee *ast.Ident
					switch fun := ast.Unparen(call.Fun).(type) {
					case *ast.Ident:
						callee = fun
					case *ast.SelectorExpr:
						callee = fun.Sel
					}
					if callee == nil || info.ObjectOf(callee) != types.Unsafe.Scope().Lookup("Offsetof") {
						continue
					}
					selection := info.Selections[selector]
					owner := selection.Recv()
					var want uint64
					for _, i := range selection.Index() {
						want += prog.OffsetOf(prog.Type(owner, llssa.GoDecl), i)
						owner = owner.Underlying().(*types.Struct).Field(i).Type()
					}
					got, ok := constant.Uint64Val(value.Value)
					if !ok || got != want {
						t.Errorf("%s: Offsetof=%v, LLVM=%d", fset.Position(call.Pos()), value.Value, want)
					}
					checked++
				}
				if checked == 0 {
					t.Fatal("no constant Offsetof expression was checked")
				}
				if strings.HasPrefix(test.name, "repeated-") {
					native := prog.Type(pkg.Types.Scope().Lookup("Native").Type(), llssa.GoDecl)
					base := prog.OffsetOf(native, 1)
					switch test.name {
					case "repeated-mixed":
						goValue := prog.Type(pkg.Types.Scope().Lookup("GoValue").Type(), llssa.GoDecl)
						base += prog.OffsetOf(goValue, 1)
					case "repeated-promoted":
						outer := prog.Type(pkg.Types.Scope().Lookup("Outer").Type(), llssa.GoDecl)
						base += prog.OffsetOf(outer, 1)
					}
					for i, name := range []string{"A", "B", "C"} {
						want := base + uint64(i)
						if test.name == "repeated-generic" {
							want = uint64(i+1) * prog.SizeOf(native)
						}
						value := pkg.Types.Scope().Lookup(name).(*types.Const).Val()
						if got, ok := constant.Uint64Val(value); !ok || got != want {
							t.Errorf("%s=%v, want %d", name, value, want)
						}
					}
				}
			})
		}
	}
}

func checkUnsafeOffsetPackageLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "dep"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod": "module example.com/offsetload\ngo 1.27\n",
		"dep/dep.go": `package dep
import "unsafe"
type G struct { F func(); Tail uintptr }
//llgo:cdecl
type C G
const NativeOffset = unsafe.Offsetof(C{}.Tail)
`,
		"input.go": `package p
import "unsafe"
import "example.com/offsetload/dep"
const GoOffset = unsafe.Offsetof(dep.G{}.Tail)
const NativeOffset = unsafe.Offsetof(dep.C{}.Tail)
var Buffer [NativeOffset]byte
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	fset := token.NewFileSet()
	dedup := packages.NewDeduper()
	preloads := 0
	dedup.SetPreload(func(pkg *packages.Package) {
		preloads++
		if err := cl.ParsePkgSyntax(prog, fset, pkg.Types, pkg.Syntax); err != nil {
			t.Fatal(err)
		}
		pkg.TypesSizes = withUnsafeOffsetOwner(prog, pkg)
	})
	dedup.SetRewrite(func(pkg *packages.Package) (bool, error) {
		if pkg.Name != "p" {
			return false, nil
		}
		file, err := parser.ParseFile(fset, "rewrite.go", `package p
const Rechecked = unsafe.Offsetof(dep.C{}.Tail)
`, 0)
		if err != nil {
			return false, err
		}
		pkg.Syntax[0].Decls = append(pkg.Syntax[0].Decls, file.Decls...)
		return true, nil
	})
	base := types.SizesFor("gc", runtime.GOARCH)
	sizes := func(sizes types.Sizes, _, _ string) types.Sizes {
		return prog.TypeSizes(sizes)
	}
	config := &packages.Config{
		Dir: dir, Fset: fset,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedSyntax,
	}
	for _, phase := range []string{"initial", "cached"} {
		pkgs, err := packages.LoadEx(dedup, sizes, config, ".")
		if err != nil || len(pkgs) != 1 {
			t.Fatalf("%s: packages=%d err=%v", phase, len(pkgs), err)
		}
		pkg := pkgs[0]
		if pkg.IllTyped {
			t.Fatalf("%s: %v", phase, pkg.Errors)
		}
		word := base.Sizeof(types.Typ[types.Uintptr])
		for name, want := range map[string]int64{"GoOffset": 2 * word, "NativeOffset": word, "Rechecked": word} {
			obj, ok := pkg.Types.Scope().Lookup(name).(*types.Const)
			if !ok {
				t.Fatalf("%s: missing constant %s", phase, name)
			}
			if got, ok := constant.Int64Val(obj.Val()); !ok || got != want {
				t.Errorf("%s: %s=%v, want %d", phase, name, obj.Val(), want)
			}
		}
		if got := pkg.Types.Scope().Lookup("Buffer").Type().Underlying().(*types.Array).Len(); got != word {
			t.Errorf("%s: array bound=%d, want %d", phase, got, word)
		}
		if preloads != 3 {
			t.Errorf("%s: preloads=%d, want dependency plus two checks of input", phase, preloads)
		}
	}
}
