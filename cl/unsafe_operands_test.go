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
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	gossa "golang.org/x/tools/go/ssa"
)

func compileRewrittenUnsafe(t *testing.T, src string) (*gossa.Package, llvm.Module) {
	return compileRewrittenUnsafeForTarget(t, src, nil)
}

func compileRewrittenUnsafeForTarget(t *testing.T, src string, target *llssa.Target, rawSSA ...bool) (*gossa.Package, llvm.Module) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Instances:  make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Implicits:  make(map[ast.Node]types.Object),
	}
	conf := types.Config{Importer: packages.NewImporter(fset)}
	checked, err := conf.Check("foo", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	if len(rawSSA) == 0 || !rawSSA[0] {
		RewriteUnsafeOperands(files, info)
		RewriteUnsafeOperands(files, info) // The package registration paths may overlap.
	}
	prog := gossa.NewProgram(fset, gossa.SanityCheckFunctions|gossa.InstantiateGenerics)
	var createImports func(*types.Package)
	createImports = func(pkg *types.Package) {
		for _, imp := range pkg.Imports() {
			if prog.Package(imp) == nil {
				prog.CreatePackage(imp, nil, nil, true)
				createImports(imp)
			}
		}
	}
	createImports(checked)
	pkg := prog.CreatePackage(checked, files, info, true)
	pkg.Build()
	llprog := newLLSSAProgForTarget(t, target)
	if target == nil {
		llprog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	}
	t.Cleanup(llprog.Dispose)
	compiled, err := NewPackage(llprog, pkg, files)
	if err != nil {
		t.Fatal(err)
	}
	module := compiled.Module()
	t.Cleanup(module.Dispose)
	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	return pkg, module
}

func TestCompileOffsetOfOperandIsNotEvaluated(t *testing.T) {
	_, module := compileRewrittenUnsafe(t, `package foo

import "unsafe"

type S[T any] struct {
	A byte
	B T
}

func pointer[T any]() *S[T] {
	panic("operand evaluated")
}

func offset[T any]() uintptr {
	return unsafe.Offsetof(pointer[T]().B)
}

func Result() uintptr { return offset[int64]() }
`)
	if strings.Contains(module.String(), "operand evaluated") {
		t.Fatal("unsafe.Offsetof operand was emitted into the compiled result")
	}
}

func TestRewriteOffsetofBuildsOnlyTypeAndPath(t *testing.T) {
	const src = `package foo
import u "unsafe"
type inner[T any] struct { Pad byte; Value T }
type box[T any] struct { Pad byte; inner[T] }
func effect[T any]() bool { panic("operand evaluated") }
func pointer[T any]() *box[T] { panic("operand evaluated") }
func offset[T any]() uintptr {
 return u.Offsetof(pointer[T]().Value) +
 u.Offsetof(struct { Pad byte; Value T; Flag bool }{Flag: effect[T]() && effect[T]()}.Value)
}
func Result() uintptr { return offset[int64]() }
`
	for _, target := range []struct {
		name string
		arch *llssa.Target
		want uint64
	}{
		{"native", nil, uint64(3 * types.SizesFor("gc", runtime.GOARCH).Alignof(types.Typ[types.Int64]))},
		{"amd64", &llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 24},
		{"386", &llssa.Target{GOOS: "windows", GOARCH: "386"}, 12},
		{"wasm32", &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, 24},
		{"wasm64", &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", WasmProvider: "emscripten"}, 24},
	} {
		t.Run(target.name, func(t *testing.T) {
			pkg, module := compileRewrittenUnsafeForTarget(t, src, target.arch)
			var offsets int
			for _, block := range pkg.Func("offset").Blocks {
				for _, instr := range block.Instrs {
					switch instr := instr.(type) {
					case *gossa.Call:
						builtin, ok := instr.Call.Value.(*gossa.Builtin)
						if !ok || builtin.Name() != "Offsetof" {
							t.Fatalf("unexpected operand call in SSA: %s", instr)
						}
						for _, arg := range instr.Call.Args {
							if _, ok := arg.(*gossa.Const); !ok {
								t.Fatalf("nonconstant Offsetof metadata: %s", arg)
							}
						}
						offsets++
					case *gossa.If, *gossa.Phi, *gossa.Alloc, *gossa.FieldAddr, *gossa.Field, *gossa.UnOp, *gossa.Store:
						t.Fatalf("operand instruction emitted into SSA: %s", instr)
					}
				}
			}
			options := llvm.NewPassBuilderOptions()
			t.Cleanup(options.Dispose)
			if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
				t.Fatal(err)
			}
			result := module.NamedFunction("foo.offset[int64]")
			if result.IsNil() {
				t.Fatal("missing instantiated offset function")
			}
			ret := result.FirstBasicBlock().LastInstruction()
			if ret.IsAReturnInst().IsNil() || ret.OperandsCount() != 1 {
				t.Fatalf("instantiated offset did not return a value: %s", result.String())
			}
			value := ret.Operand(0).IsAConstantInt()
			if value.IsNil() || value.ZExtValue() != target.want {
				t.Fatalf("instantiated offset = %s, want %d", ret.String(), target.want)
			}
			if offsets != 2 {
				t.Fatalf("Offsetof calls = %d, want 2", offsets)
			}
			if strings.Contains(module.String(), "operand evaluated") {
				t.Fatal("operand body emitted into LLVM IR")
			}
		})
	}
}

func TestRewriteUnsafeImportForms(t *testing.T) {
	for _, builtin := range []string{"Offsetof", "Sizeof", "Alignof"} {
		for _, form := range []struct {
			name, imp, call string
		}{
			{"qualified", `"unsafe"`, "unsafe.Offsetof"},
			{"alias", `u "unsafe"`, "u.Offsetof"},
			{"dot", `. "unsafe"`, "Offsetof"},
		} {
			t.Run(builtin+"/"+form.name, func(t *testing.T) {
				src := `package foo
import IMPORT
func pointer[T any]() *struct { Pad byte; Value T } { panic("operand evaluated") }
func offset[T any]() uintptr { return (BUILTIN)((pointer[T]().Value)) }
func Result() uintptr { return offset[int64]() }
`
				src = strings.ReplaceAll(src, "IMPORT", form.imp)
				src = strings.ReplaceAll(src, "BUILTIN", strings.ReplaceAll(form.call, "Offsetof", builtin))
				_, module := compileRewrittenUnsafe(t, src)
				if !strings.Contains(module.String(), "ret i64 8") {
					t.Fatalf("rewritten builtin result is missing: %s", module.String())
				}
				if strings.Contains(module.String(), `call void @"foo.pointer`) {
					t.Fatal("operand call was emitted")
				}
			})
		}

	}
}

func TestRewriteSizeofAlignofTypeOnly(t *testing.T) {
	for _, builtin := range []string{"Sizeof", "Alignof"} {
		for _, target := range []struct {
			name            string
			arch            *llssa.Target
			size, alignment uint64
		}{
			{"amd64", &llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 8, 8},
			{"386", &llssa.Target{GOOS: "windows", GOARCH: "386"}, 8, 8},
			{"wasm32", &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, 8, 8},
			{"wasm64", &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", WasmProvider: "emscripten"}, 8, 8},
		} {
			t.Run(builtin+"/"+target.name, func(t *testing.T) {
				src := `package foo
import . "unsafe"
func operand[T any]() T { panic("operand evaluated") }
func layout[T any]() uintptr { return (BUILTIN)(operand[T]()) }
func Result() uintptr { return layout[int64]() }
`
				src = strings.ReplaceAll(src, "BUILTIN", builtin)
				pkg, module := compileRewrittenUnsafeForTarget(t, src, target.arch)
				calls := 0
				for _, block := range pkg.Func("layout").Blocks {
					for _, instr := range block.Instrs {
						switch instr := instr.(type) {
						case *gossa.Call:
							fn, ok := instr.Call.Value.(*gossa.Builtin)
							if !ok || fn.Name() != builtin || len(instr.Call.Args) != 2 {
								t.Fatalf("unexpected call: %s", instr)
							}
							for _, arg := range instr.Call.Args {
								if _, ok := arg.(*gossa.Const); !ok {
									t.Fatalf("nonconstant metadata: %s", arg)
								}
							}
							calls++
						case *gossa.Return:
						default:
							t.Fatalf("unexpected operand instruction: %s", instr)
						}
					}
				}
				if calls != 1 {
					t.Fatalf("layout calls = %d, want 1", calls)
				}
				options := llvm.NewPassBuilderOptions()
				t.Cleanup(options.Dispose)
				if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
					t.Fatal(err)
				}
				fn := module.NamedFunction("foo.layout[int64]")
				if fn.IsNil() {
					t.Fatal("missing layout instance")
				}
				ret := fn.FirstBasicBlock().LastInstruction()
				want := target.size
				if builtin == "Alignof" {
					want = target.alignment
				}
				value := ret.Operand(0).IsAConstantInt()
				if value.IsNil() || value.ZExtValue() != want {
					t.Fatalf("%s, want %d", ret.String(), want)
				}
				if strings.Contains(module.String(), "operand evaluated") {
					t.Fatal("operand emitted into LLVM IR")
				}
			})
		}
	}
}

func TestRewriteUnsafeNamedLayout(t *testing.T) {
	const src = `package foo
import "unsafe"
type GoRecord[T any] struct { Ptr *byte; Tail T }
//llgo:cdecl
type NativeFromGo[T any] GoRecord[T]
//llgo:cdecl
type NativeRecord[T any] struct { Ptr *byte; Tail T }
type GoFromNative[T any] NativeRecord[T]
type NativeWrapper[T any] struct { Pad byte; NativeFromGo[T] }
type GoWrapper[T any] struct { Pad byte; GoFromNative[T] }
func nativeSize[T any]() uintptr { return unsafe.Sizeof(NativeFromGo[T]{}) }
func nativeAlign[T any]() uintptr { return unsafe.Alignof(NativeFromGo[T]{}) }
func goSize[T any]() uintptr { return unsafe.Sizeof(GoFromNative[T]{}) }
func goAlign[T any]() uintptr { return unsafe.Alignof(GoFromNative[T]{}) }
func nativeOffset[T any]() uintptr { var p *NativeFromGo[T]; return unsafe.Offsetof(p.Tail) }
func goOffset[T any]() uintptr { var p *GoFromNative[T]; return unsafe.Offsetof(p.Tail) }
func nativeArray[T any]() uintptr { return unsafe.Sizeof([2]NativeFromGo[T]{}) }
func goArray[T any]() uintptr { return unsafe.Sizeof([2]GoFromNative[T]{}) }
func nativePromoted[T any]() uintptr { var p *NativeWrapper[T]; return unsafe.Offsetof(p.Tail) }
func goPromoted[T any]() uintptr { var p *GoWrapper[T]; return unsafe.Offsetof(p.Tail) }
func Result() uintptr {
 return nativeSize[uint32]() + nativeAlign[uint32]() + goSize[uint32]() + goAlign[uint32]() +
 nativeOffset[uint32]() + goOffset[uint32]() + nativeArray[uint32]() + goArray[uint32]() +
 nativePromoted[uint32]() + goPromoted[uint32]()
}
`
	for _, tc := range []struct {
		name               string
		target             llssa.Target
		nativeWord, goWord uint64
	}{
		{"wasi32", llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}, 4, 8},
		{"emscripten32", llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"}, 4, 8},
		{"emscripten64", llssa.Target{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64", WasmProvider: "emscripten"}, 8, 8},
		{"amd64", llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 8, 8},
		{"386", llssa.Target{GOOS: "windows", GOARCH: "386"}, 4, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, module := compileRewrittenUnsafeForTarget(t, src, &tc.target)
			options := llvm.NewPassBuilderOptions()
			t.Cleanup(options.Dispose)
			if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]uint64{
				"nativeSize": 2 * tc.nativeWord, "nativeAlign": tc.nativeWord,
				"goSize": 2 * tc.goWord, "goAlign": tc.goWord,
				"nativeOffset": tc.nativeWord, "goOffset": tc.goWord,
				"nativeArray": 4 * tc.nativeWord, "goArray": 4 * tc.goWord,
				"nativePromoted": 2 * tc.nativeWord, "goPromoted": tc.nativeWord + tc.goWord,
			} {
				fn := module.NamedFunction("foo." + name + "[uint32]")
				if fn.IsNil() {
					t.Fatalf("missing instance %s", name)
				}
				ret := fn.FirstBasicBlock().LastInstruction()
				if ret.IsAReturnInst().IsNil() || ret.OperandsCount() != 1 {
					t.Fatalf("%s: %s", name, fn.String())
				}
				value := ret.Operand(0).IsAConstantInt()
				if value.IsNil() || value.ZExtValue() != want {
					t.Errorf("%s: %s, want %d", name, ret.String(), want)
				}
			}
		})
	}
}

func TestRewriteUnsafeCallbackStorage(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Record[T any] struct { Fn func(); Tail T }
//llgo:cdecl
type Callback func()
STDCALL
type StdCallback func()
type Controls[T any] struct { C Callback; Std StdCallback; Tail T }
type Wrapper[T any] struct { Pad byte; Record[T] }
type Zero[T any] struct { Value T; End [0]byte }
func offset[T any]() uintptr { var p *Record[T]; return unsafe.Offsetof(p.Tail) }
func promoted[T any]() uintptr { var p *Wrapper[T]; return unsafe.Offsetof(p.Tail) }
func explicit[T any]() uintptr { var p *Wrapper[T]; return unsafe.Offsetof(p.Record.Tail) }
func controls[T any]() uintptr { var p *Controls[T]; return unsafe.Offsetof(p.Tail) }
func zero[T any]() uintptr { var p *Zero[T]; return unsafe.Sizeof(*p) }
func Result() uintptr { return offset[int32]()+promoted[int32]()+explicit[int32]()+controls[int32]()+zero[uint64]() }
`
	for _, tc := range []struct {
		name   string
		target llssa.Target
		word   uint64
	}{
		{"arm64", llssa.Target{GOOS: "darwin", GOARCH: "arm64"}, 8},
		{"amd64", llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 8},
		{"windows386", llssa.Target{GOOS: "windows", GOARCH: "386"}, 4},
		{"windowsamd64", llssa.Target{GOOS: "windows", GOARCH: "amd64"}, 8},
		{"windowsarm64", llssa.Target{GOOS: "windows", GOARCH: "arm64"}, 8},
		{"wasm64", llssa.Target{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64", WasmProvider: "emscripten"}, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directive := "//llgo:cdecl"
			if tc.target.GOOS == "windows" {
				directive = "//llgo:stdcall"
			}
			_, module := compileRewrittenUnsafeForTarget(t, strings.ReplaceAll(src, "STDCALL", directive), &tc.target)
			options := llvm.NewPassBuilderOptions()
			t.Cleanup(options.Dispose)
			if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]uint64{"offset": tc.word, "explicit": tc.word, "promoted": 2 * tc.word, "controls": 2 * tc.word, "zero": 8} {
				arg := "int32"
				if name == "zero" {
					arg = "uint64"
					if tc.name == "windows386" {
						want = 12
					}
					if tc.name == "wasm64" {
						want = 16
					}
				}
				fn := module.NamedFunction("foo." + name + "[" + arg + "]")
				if fn.IsNil() {
					t.Fatalf("missing %s", name)
				}
				ret := fn.FirstBasicBlock().LastInstruction()
				value := ret.Operand(0).IsAConstantInt()
				if value.IsNil() || value.ZExtValue() != want {
					t.Errorf("%s: %s, want %d", name, ret.String(), want)
				}
			}
		})
	}
}

func TestRewriteUnsafeNativeAlignmentAndEmbeddedZero(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Native[T any] struct { Pad byte; Value T }
type Ordinary[T any] struct { Pad byte; Value T }
type Zero[T any] struct { Value T; End [0]byte }
type Outer[T any] struct { Zero[T]; Tail uint64 }
func alignment[T any](p *T) uintptr { return unsafe.Alignof(*p) }
func offset[T any](p *Outer[T]) uintptr { return unsafe.Offsetof(p.Tail) }
func Result() uintptr {
 var n Native[uint64]; var a [2]Native[uint64]
 var g Ordinary[uint64]; var ga [2]Ordinary[uint64]; var o Outer[uint64]
 return alignment(&n)+alignment(&a)+alignment(&g)+alignment(&ga)+offset(&o)
}
`
	for _, tc := range []struct {
		name             string
		target           llssa.Target
		ordinary, offset uint64
	}{
		{"darwinarm64", llssa.Target{GOOS: "darwin", GOARCH: "arm64"}, 8, 8},
		{"linuxamd64", llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 8, 8},
		{"windows386", llssa.Target{GOOS: "windows", GOARCH: "386"}, 4, 12},
		{"wasi", llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}, 8, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, module := compileRewrittenUnsafeForTarget(t, src, &tc.target)
			options := llvm.NewPassBuilderOptions()
			t.Cleanup(options.Dispose)
			if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]uint64{
				"alignment[foo.Native[uint64]]":      8,
				"alignment[[2]foo.Native[uint64]]":   8,
				"alignment[foo.Ordinary[uint64]]":    tc.ordinary,
				"alignment[[2]foo.Ordinary[uint64]]": tc.ordinary,
				"offset[uint64]":                     tc.offset,
			} {
				fn := module.NamedFunction("foo." + name)
				if fn.IsNil() {
					t.Fatalf("missing %s\n%s", name, module.String())
				}
				ret := fn.FirstBasicBlock().LastInstruction()
				value := ret.Operand(0).IsAConstantInt()
				if value.IsNil() || value.ZExtValue() != want {
					t.Errorf("%s: %s, want %d", name, ret.String(), want)
				}
			}
		})
	}
}

func TestRewriteUnsafeContainingNativeAlignment(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Native[T any] struct { Pad byte; Value T }
type Outer[T any] struct { Prefix byte; Native[T]; Tail byte }
type ArrayOuter[T any] struct { Prefix byte; Values [2]Native[T]; Tail byte }
type Nested[T any] struct { Prefix byte; Outer[T]; Tail byte }
type GoOnly[T any] struct { Prefix byte; Value T }
func alignment[T any](p *T) uintptr { return unsafe.Alignof(*p) }
func embedded[T any](p *Outer[T]) uintptr { return unsafe.Offsetof(p.Native) }
func promoted[T any](p *Outer[T]) uintptr { return unsafe.Offsetof(p.Value) }
func tail[T any](p *Outer[T]) uintptr { return unsafe.Offsetof(p.Tail) }
func arrayField[T any](p *ArrayOuter[T]) uintptr { return unsafe.Offsetof(p.Values) }
func arrayTail[T any](p *ArrayOuter[T]) uintptr { return unsafe.Offsetof(p.Tail) }
func nested[T any](p *Nested[T]) uintptr { return unsafe.Offsetof(p.Value) }
func Result() uintptr {
 var o Outer[uint64]; var a ArrayOuter[uint64]; var n Nested[uint64]; var g GoOnly[uint64]; var arr [2]Outer[uint64]
 return alignment(&o)+alignment(&a)+alignment(&n)+alignment(&g)+alignment(&arr)+embedded(&o)+promoted(&o)+tail(&o)+arrayField(&a)+arrayTail(&a)+nested(&n)
}
`
	for _, tc := range []struct {
		name    string
		target  llssa.Target
		goAlign uint64
	}{
		{"windows386", llssa.Target{GOOS: "windows", GOARCH: "386"}, 4},
		{"arm64", llssa.Target{GOOS: "darwin", GOARCH: "arm64"}, 8},
		{"amd64", llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, module := compileRewrittenUnsafeForTarget(t, src, &tc.target)
			options := llvm.NewPassBuilderOptions()
			t.Cleanup(options.Dispose)
			if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]uint64{
				"alignment[foo.Outer[uint64]]": 8, "alignment[foo.ArrayOuter[uint64]]": 8,
				"alignment[foo.Nested[uint64]]": 8, "alignment[[2]foo.Outer[uint64]]": 8,
				"alignment[foo.GoOnly[uint64]]": tc.goAlign,
				"embedded[uint64]":              8, "promoted[uint64]": 16, "tail[uint64]": 24,
				"arrayField[uint64]": 8, "arrayTail[uint64]": 40, "nested[uint64]": 24,
			} {
				if tc.name == "windows386" {
					if name == "nested[uint64]" {
						want = 20
					}
					if name == "alignment[foo.Nested[uint64]]" {
						want = 4
					}
				}
				fn := module.NamedFunction("foo." + name)
				if fn.IsNil() {
					t.Fatalf("missing %s", name)
				}
				ret := fn.FirstBasicBlock().LastInstruction()
				value := ret.Operand(0).IsAConstantInt()
				if value.IsNil() || value.ZExtValue() != want {
					t.Errorf("%s: %s, want %d", name, ret.String(), want)
				}
			}
		})
	}
}

func TestRewriteUnsafePreservesExistingLayout(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Native[T any] struct { Pad byte; Value T }
type Outer[T any] struct { Prefix byte; Native[T]; Tail byte }
type Nested[T any] struct { Prefix byte; Outer[T]; Tail byte }
func scalar[T any](v T) uintptr { return unsafe.Alignof(v) }
func nested[T any](v Nested[T]) uintptr { return unsafe.Alignof(v) }
func offset[T any]() uintptr { var v Outer[T]; return unsafe.Offsetof(v.Value) }
func Result() uintptr { return scalar(uint64(0))+nested(Nested[uint64]{})+offset[uint64]() }
`
	target := &llssa.Target{GOOS: "windows", GOARCH: "386"}
	results := make([]map[string]uint64, 2)
	for i, raw := range []bool{true, false} {
		_, module := compileRewrittenUnsafeForTarget(t, src, target, raw)
		opts := llvm.NewPassBuilderOptions()
		t.Cleanup(opts.Dispose)
		if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, opts); err != nil {
			t.Fatal(err)
		}
		results[i] = make(map[string]uint64)
		for _, name := range []string{"scalar[uint64]", "nested[uint64]", "offset[uint64]"} {
			fn := module.NamedFunction("foo." + name)
			if fn.IsNil() {
				t.Fatalf("missing %s", name)
			}
			ret := fn.FirstBasicBlock().LastInstruction()
			if ret.IsAReturnInst().IsNil() {
				t.Fatalf("%s: %s", name, fn.String())
			}
			value := ret.Operand(0).IsAConstantInt()
			if value.IsNil() {
				t.Fatalf("%s: %s", name, ret.String())
			}
			results[i][name] = value.ZExtValue()
		}
	}
	for name, want := range results[0] {
		if got := results[1][name]; got != want {
			t.Errorf("%s: rewritten %d, original SSA %d", name, got, want)
		}
	}
}

func TestRewriteUnsafeNativeFieldRepresentation(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Native[T any] struct { Value T; Tail byte }
type Embedded[T any] struct { *Native[T] }
func pointer[T any]() *Native[T] { panic("operand evaluated") }
func field[T any](p *Native[T]) uintptr { return unsafe.Sizeof(p.Value) }
func value[T any](p Native[T]) uintptr { return unsafe.Sizeof(p.Value) }
func promoted[T any](p *Embedded[T]) uintptr { return unsafe.Sizeof(p.Value) }
func alignment[T any](p *Native[T]) uintptr { return unsafe.Alignof(p.Value) }
func effect[T any]() uintptr { return unsafe.Sizeof(pointer[T]().Value) }
func Result() uintptr {
 var n Native[func()];var a Native[[2]func()];var e Embedded[func()]
 return field(&n)+field(&a)+value(n)+promoted(&e)+alignment(&n)+effect[func()]()
}
`
	for _, tc := range []struct {
		name   string
		target llssa.Target
		word   uint64
	}{
		{"arm64", llssa.Target{GOOS: "darwin", GOARCH: "arm64"}, 8},
		{"amd64", llssa.Target{GOOS: "linux", GOARCH: "amd64"}, 8},
		{"windows386", llssa.Target{GOOS: "windows", GOARCH: "386"}, 4},
		{"wasi", llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, module := compileRewrittenUnsafeForTarget(t, src, &tc.target)
			opts := llvm.NewPassBuilderOptions()
			t.Cleanup(opts.Dispose)
			if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, opts); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]uint64{
				"field[func()]": tc.word, "field[[2]func()]": 2 * tc.word,
				"value[func()]": tc.word, "promoted[func()]": tc.word, "alignment[func()]": tc.word, "effect[func()]": tc.word,
			} {
				fn := module.NamedFunction("foo." + name)
				if fn.IsNil() {
					t.Fatalf("missing %s", name)
				}
				ret := fn.FirstBasicBlock().LastInstruction()
				v := ret.Operand(0).IsAConstantInt()
				if v.IsNil() || v.ZExtValue() != want {
					t.Errorf("%s: %s, want %d", name, ret.String(), want)
				}
			}
			if strings.Contains(module.String(), "operand evaluated") {
				t.Fatal("operand was emitted")
			}
		})
	}
}

func TestRewriteUnsafeSkipsNonGenericOperands(t *testing.T) {
	const src = `package foo
import "unsafe"
type S struct { Value int }
func pointer() *S { panic("non-generic operand evaluated") }
func Result() uintptr { return unsafe.Sizeof(pointer().Value) }
`
	_, module := compileRewrittenUnsafe(t, src)
	if !strings.Contains(module.String(), "non-generic operand evaluated") {
		t.Fatal("non-generic unsafe operand was unexpectedly rewritten")
	}
}

func TestRewriteUnsafeOperandRepresentationPath(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Native[T any] struct { Value T; Items [2]T }
func field[T any](p *Native[T]) uintptr { return unsafe.Sizeof(p.Value) }
func alignment[T any](p *Native[T]) uintptr { return unsafe.Alignof(p.Value) }
func element[T any](p *Native[T], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func value[T any](p Native[T], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func deref[T any](p *Native[T]) uintptr { return unsafe.Sizeof(*(&p.Value)) }
func Result() uintptr {
 var f Native[func()];var u Native[uintptr]
 return field(&f)+alignment(&f)+element(&f,1)+value(f,0)+deref(&f)+field(&u)+alignment(&u)+element(&u,1)+deref(&u)
}
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			results := make([]map[string]uint64, 2)
			for i, raw := range []bool{true, false} {
				_, module := compileRewrittenUnsafeForTarget(t, src, &target, raw)
				opts := llvm.NewPassBuilderOptions()
				t.Cleanup(opts.Dispose)
				if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, opts); err != nil {
					t.Fatal(err)
				}
				results[i] = make(map[string]uint64)
				for _, name := range []string{"field[func()]", "alignment[func()]", "element[func()]", "value[func()]", "deref[func()]", "field[uintptr]", "alignment[uintptr]", "element[uintptr]", "deref[uintptr]"} {
					fn := module.NamedFunction("foo." + name)
					if fn.IsNil() {
						t.Fatalf("missing %s", name)
					}
					found := false
					for _, block := range fn.BasicBlocks() {
						ret := block.LastInstruction()
						if ret.IsAReturnInst().IsNil() {
							continue
						}
						v := ret.Operand(0).IsAConstantInt()
						if v.IsNil() {
							t.Fatalf("%s: %s", name, ret.String())
						}
						results[i][name] = v.ZExtValue()
						found = true
					}
					if !found {
						t.Fatalf("missing return: %s", name)
					}
				}
			}
			for name, want := range results[0] {
				if got := results[1][name]; got != want {
					t.Errorf("%s: rewritten %d, raw %d", name, got, want)
				}
			}
		})
	}
}

func TestRewriteUnsafeWindows386OperandExpectations(t *testing.T) {
	const src = `package foo
import "unsafe"
func operand[T any]() T { panic("operand evaluated") }
func scalar[T any]() bool {
 var reference T
 size, alignment := unsafe.Sizeof(reference), unsafe.Alignof(reference)
 return unsafe.Sizeof(operand[T]()) == size && unsafe.Alignof(operand[T]()) == alignment
}
func wrapped[T any]() bool {
 var reference struct { Value T }
 size, alignment := unsafe.Sizeof(reference), unsafe.Alignof(reference)
 return unsafe.Sizeof(struct { Value T }{operand[T]()}) == size && unsafe.Alignof(struct { Value T }{operand[T]()}) == alignment
}
func nilOperand[T any]() bool {
 var p *T
 var reference T
 size, alignment := unsafe.Sizeof(reference), unsafe.Alignof(reference)
 return unsafe.Sizeof(*p) == size && unsafe.Alignof(*p) == alignment
}
func Result() bool { return scalar[int64]() && wrapped[int64]() && nilOperand[int64]() }
`
	_, module := compileRewrittenUnsafeForTarget(t, src, &llssa.Target{GOOS: "windows", GOARCH: "386"})
	options := llvm.NewPassBuilderOptions()
	t.Cleanup(options.Dispose)
	if err := module.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"scalar", "wrapped", "nilOperand"} {
		fn := module.NamedFunction("foo." + name + "[int64]")
		if fn.IsNil() {
			t.Fatalf("missing %s", name)
		}
		ret := fn.FirstBasicBlock().LastInstruction()
		if ret.IsAReturnInst().IsNil() || ret.OperandsCount() != 1 {
			t.Fatalf("unexpected body: %s", fn.String())
		}
		value := ret.Operand(0).IsAConstantInt()
		if value.IsNil() || value.ZExtValue() != 1 {
			t.Errorf("Windows/386 %s comparison failed: %s", name, fn.String())
		}
	}
	if strings.Contains(module.String(), "operand evaluated") {
		t.Fatal("operand emitted into Windows/386 module")
	}
}
