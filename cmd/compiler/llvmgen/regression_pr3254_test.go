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
	"go/build"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"

	llpkgs "github.com/xgo-dev/llgo/cmd/compiler/packages"
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

func TestNativeCallConvPreservedAcrossNativeAggregateInit(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"})
	defer prog.Dispose()
	prog.SetRuntime(newDebugRuntimePackage())
	pkg := prog.NewPackage("natcc", "example.com/natcc")

	// Register a CDecl FuncDecl first (populates nativeCC while nativeViews is still nil).
	retTuple := types.NewTuple(
		types.NewParam(token.NoPos, nil, "", NoArgsNoRet),
		types.NewParam(token.NoPos, nil, "", types.Typ[types.Bool]),
	)
	calleeSig := types.NewSignatureType(nil, nil, nil, nil, retTuple, false)
	callee := pkg.NewFunc("c_multi_ret_func", calleeSig, CDecl)

	// Now initialize nativeViews via a native aggregate type; this must NOT wipe nativeCC.
	arrType := prog.Type(types.NewArray(NoArgsNoRet, 2), CDecl)
	if arrType.kind != vkArray {
		t.Fatalf("native array kind = %v, want vkArray", arrType.kind)
	}
	if cc := prog.nativeCallConvOf(callee.Type); cc != CDecl {
		t.Fatalf("callee nativeCC after native aggregate init = %v, want CDecl", cc)
	}

	callerSig := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.Bool])), false)
	caller := pkg.NewFunc("caller", callerSig, GoDecl)
	b := caller.MakeBody(1)
	defer b.Dispose()
	res := b.Call(callee.Expr)
	fnVal := b.Extract(res, 0)
	if fnVal.kind != vkFuncPtr {
		t.Fatalf("extracted native function result kind = %v, want vkFuncPtr", fnVal.kind)
	}
	okVal := b.Extract(res, 1)
	b.Return(okVal)
	b.EndBuild()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid amd64 native multi-return module: %v\n%s", err, pkg.Module().String())
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

func TestPublicReflectHigherOrderNativeFuncAliasBridge(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	prog := NewProgram(&Target{
		GOOS:               "js",
		GOARCH:             "wasm",
		WasmProfile:        "j32",
		WasmProvider:       "emscripten",
		LLVMTarget:         "wasm32-unknown-emscripten",
		WasmReflectBridges: true,
	})
	defer prog.Dispose()
	buildContext := build.Default
	build.Default.GOOS = "wasip1"
	build.Default.GOARCH = "wasm"
	build.Default.BuildTags = []string{"llgo", "llgo.wasi_threads"}
	t.Cleanup(func() { build.Default = buildContext })
	runtimePkg, err := llpkgs.SourceImporter().Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	prog.SetRuntime(runtimePkg)
	pkg := prog.NewPackage("hoalias", "example.com/hoalias")

	const src = `package hoalias

type cbAlias = func(uintptr) string
type higher = func(cbAlias) cbAlias
type expanded = func(func(uintptr) string) func(uintptr) string
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "hoalias.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	tpkg, err := (&types.Config{Context: prog.typeCtx}).Check("example.com/hoalias", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prog.typeCtx.SetTypeCallConv("example.com/hoalias.cbAlias", types.CDecl)

	cbAlias := tpkg.Scope().Lookup("cbAlias").Type()
	higher := tpkg.Scope().Lookup("higher").Type()
	expanded := tpkg.Scope().Lookup("expanded").Type()

	pubCB := prog.publicReflectFunc(cbAlias)
	if pubCB == nil || pubCB.CallConv() != types.Go {
		t.Fatalf("publicReflectFunc(cbAlias) = %v (cc=%v), want Go signature", pubCB, pubCB.CallConv())
	}
	pubHigher := prog.publicReflectFunc(higher)
	if pubHigher == nil {
		t.Fatal("publicReflectFunc(higher) = nil, want projected Go signature")
	}
	if paramSig, ok := pubHigher.Params().At(0).Type().(*types.Signature); !ok || paramSig.CallConv() != types.Go {
		t.Fatalf("publicReflectFunc(higher) param0 = %T (cc=%v), want Go *Signature", pubHigher.Params().At(0).Type(), paramSig.CallConv())
	}
	if resSig, ok := pubHigher.Results().At(0).Type().(*types.Signature); !ok || resSig.CallConv() != types.Go {
		t.Fatalf("publicReflectFunc(higher) result0 = %T (cc=%v), want Go *Signature", pubHigher.Results().At(0).Type(), resSig.CallConv())
	}

	// Emit the higher-order alias descriptor BEFORE the expanded public descriptor
	// so the public descriptor's Wasm reflect bridge is materialized via publicSig.
	fn := pkg.NewFunc("emit", NoArgsNoRet, GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()
	higherDesc := b.abiPublicType(higher).impl
	expandedDesc := b.abiPublicType(expanded).impl
	b.Return()
	b.EndBuild()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid module: %v\n%s", err, pkg.Module().String())
	}
	// The public descriptor emitted via higher's publicSig must carry the 2-word
	// Go closure Wasm reflect bridges (matching expanded), not 1-word native ptr bridges.
	if gotPub := higherDesc.Initializer().Operand(1).Operand(0); gotPub != expandedDesc {
		t.Fatalf("higher-order alias descriptor points to wrong public type: %s != %s", gotPub, expandedDesc)
	}
	expandedSig := types.Unalias(expanded).(*types.Signature)
	wantBridge := pkg.wasmReflectBridge(expandedSig)
	pubBridge := pkg.wasmReflectBridge(pubHigher)
	if pubBridge.call.impl != wantBridge.call.impl || pubBridge.make.impl != wantBridge.make.impl {
		t.Fatalf("projected higher-order public signature selected different Wasm reflect bridges from expanded Go signature")
	}
	// Fields 3 and 4 of expandedDesc's FuncType initializer are the Wasm call/make bridges.
	expandedInit := expandedDesc.Initializer()
	if gotCall := expandedInit.Operand(3).Operand(0); gotCall != wantBridge.call.impl {
		t.Fatalf("expanded descriptor call bridge = %s, want %s", gotCall, wantBridge.call.impl)
	}
	if gotMake := expandedInit.Operand(4).Operand(0); gotMake != wantBridge.make.impl {
		t.Fatalf("expanded descriptor make bridge = %s, want %s", gotMake, wantBridge.make.impl)
	}
}
