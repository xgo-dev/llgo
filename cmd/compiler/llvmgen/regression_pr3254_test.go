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

package llvmgen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
)

func TestSliceUncheckedArrayPointerElementStride(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"})
	defer prog.Dispose()
	prog.SetRuntime(newDebugRuntimePackage())
	pkg := prog.NewPackage("slicearr", "example.com/slicearr")

	arrType := types.NewArray(types.Typ[types.Int64], 16)
	sliceType := types.NewSlice(types.Typ[types.Int64])
	sig := types.NewSignatureType(nil, nil, nil,
		nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", sliceType)),
		false,
	)
	fn := pkg.NewFunc("sliceArr", sig, GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()

	arrPtr := b.AllocaT(prog.rawType(arrType))
	res := b.SliceEx(arrPtr, prog.IntVal(2, prog.Int()), prog.IntVal(8, prog.Int()), Expr{}, false)
	b.Return(res)
	b.EndBuild()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid module: %v\n%s", err, pkg.Module().String())
	}
	ir := fn.impl.String()
	if strings.Contains(ir, "getelementptr [16 x i64]") || strings.Contains(ir, "getelementptr inbounds [16 x i64]") {
		t.Fatalf("unchecked array slice GEP strided by whole array instead of element:\n%s", ir)
	}
	if !strings.Contains(ir, "getelementptr i64, ptr") && !strings.Contains(ir, "getelementptr inbounds i64, ptr") {
		t.Fatalf("unchecked array slice GEP missing element-strided i64 pointer advance:\n%s", ir)
	}
}

func TestWasmReflectBridgeMethodAndNativeFuncPtr(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	targets := []*Target{
		{
			GOOS:               "js",
			GOARCH:             "wasm",
			WasmProfile:        "j32",
			WasmProvider:       "emscripten",
			LLVMTarget:         "wasm32-unknown-emscripten",
			WasmReflectBridges: true,
		},
		{
			GOOS:               "wasip1",
			GOARCH:             "wasm",
			WasmProfile:        "w32",
			WasmProvider:       "wasi",
			WasmReflectBridges: true,
		},
	}
	for _, target := range targets {
		t.Run(target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("wbridge", "example.com/wbridge")

			// Case 1: method signature with non-nil receiver must strip Recv() before
			// computing native integer attributes or Go/C prototypes.
			recvPkg := types.NewPackage("example.com/wbridge", "wbridge")
			recvType := types.NewNamed(types.NewTypeName(token.NoPos, recvPkg, "T", nil), types.NewStruct(nil, nil), nil)
			recv := types.NewVar(token.NoPos, recvPkg, "t", types.NewPointer(recvType))
			methodSig := types.NewSignatureType(recv, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, recvPkg, "x", types.Typ[types.Int32])),
				types.NewTuple(types.NewParam(token.NoPos, recvPkg, "", types.Typ[types.Bool])),
				false,
			)
			pkg.wasmReflectBridge(methodSig)

			// Case 2: function signature taking a callback func(uintptr) unsafe.Pointer
			// must preserve CDecl across Convert(prog.Pointer(nativeType), slot) on Wasm32.
			cbSig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr])),
				types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.UnsafePointer])),
				false,
			)
			workerSig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "cb", cbSig)),
				nil,
				false,
			)
			pkg.wasmReflectBridge(workerSig)

			// Case 3: multi-return signatures containing Go function values (such as
			// encoding/json/v2's (string, func(), func()) and (func(), bool) signatures)
			// must keep native CDecl result tuple types aligned with ll.ReturnType() on Wasm32.
			for _, results := range [][]*types.Var{
				{
					types.NewParam(token.NoPos, nil, "", types.Typ[types.String]),
					types.NewParam(token.NoPos, nil, "", NoArgsNoRet),
					types.NewParam(token.NoPos, nil, "", NoArgsNoRet),
				},
				{
					types.NewParam(token.NoPos, nil, "", NoArgsNoRet),
					types.NewParam(token.NoPos, nil, "", types.Typ[types.Bool]),
				},
				{
					types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr]),
					types.NewParam(token.NoPos, nil, "", cbSig),
					types.NewParam(token.NoPos, nil, "", types.Typ[types.Bool]),
				},
			} {
				pkg.wasmReflectBridge(types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(results...), false))
			}

			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid wasm reflect bridge module: %v\n%s", err, pkg.Module().String())
			}
		})
	}
}

func TestRuntimePkgDoesNotTriggerLazyImport(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	called := false
	prog.SetRuntime(func() *types.Package {
		called = true
		return nil
	})
	if got := prog.RuntimePkg(); got != nil {
		t.Fatalf("RuntimePkg() = %v, want nil before runtime is loaded", got)
	}
	if called {
		t.Fatal("RuntimePkg() eagerly invoked lazy runtime importer")
	}
	rt := newDebugRuntimePackage()
	prog.SetRuntime(rt)
	if got := prog.RuntimePkg(); got != rt {
		t.Fatalf("RuntimePkg() = %v, want %v", got, rt)
	}
}

func TestDebugClosureStructNameMatchesSchema(t *testing.T) {
	const src = `package p

type MyFunc func(int) int

type Holder struct {
	Plain func(int) int
	Named MyFunc
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "closure.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	typesPkg, err := (&types.Config{}).Check("example.com/p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}

	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.SetRuntime(newDebugRuntimePackage())
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", fset)

	holder := typesPkg.Scope().Lookup("Holder").Type()
	global := pkg.NewVar("example.com/p.GlobalHolder", types.NewPointer(holder), GoDecl)
	fn := pkg.NewFunc("debugClosure", NoArgsNoRet, GoDecl)
	b := fn.NewBuilder()
	defer b.impl.Dispose()
	b.DIGlobal(global.Expr, "GlobalHolder", fset.Position(typesPkg.Scope().Lookup("Holder").Pos()))
	pkg.FinalizeDebug()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid debug module: %v\n%s", err, pkg.Module().String())
	}
	ir := pkg.Module().String()
	// Matches debugabi/schema_v1.json function.type_pattern.
	wantName := `struct{$f func(int) int; $data unsafe.Pointer}`
	if !strings.Contains(ir, `name: "`+wantName+`"`) {
		t.Fatalf("closure DWARF struct name missing %q:\n%s", wantName, ir)
	}
	pat := regexp.MustCompile(`^struct\{\$f func.*; \$data unsafe\.Pointer\}$`)
	if !pat.MatchString(wantName) {
		t.Fatalf("closure name %q does not match schema pattern", wantName)
	}
	if !strings.Contains(ir, `DIDerivedType(tag: DW_TAG_typedef, name: "example.com/p.MyFunc"`) {
		t.Fatalf("named function typedef missing:\n%s", ir)
	}
}
