package ssa

import (
	"fmt"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func intrinsicDeclarationPackage(t *testing.T, target *Target) Package {
	t.Helper()
	Initialize(InitAll)
	prog := NewProgram(target)
	pkg := prog.NewPackage("intrinsicdecl", "intrinsic/decl")
	t.Cleanup(func() { pkg.Module().Dispose(); prog.Dispose() })
	return pkg
}

func buildIntrinsicTrapCaller(pkg Package, name string, callee Function) {
	b := pkg.NewFunc(name, NoArgsNoRet, CDecl).MakeBody(1)
	if callee == nil {
		b.impl.CreateIntrinsic(pkg.Prog.Void().ll, llvm.LookupIntrinsicID("llvm.trap"), nil, "")
	} else {
		b.Call(callee.Expr)
	}
	b.Unreachable()
	b.EndBuild()
	b.Dispose()
}

func TestLLVMIntrinsicDeclarationOrder(t *testing.T) {
	for _, target := range []struct {
		name string
		arch *Target
	}{
		{"native", nil},
		{"linux-amd64", &Target{GOOS: "linux", GOARCH: "amd64"}},
		{"darwin-arm64", &Target{GOOS: "darwin", GOARCH: "arm64"}},
		{"windows-amd64", &Target{GOOS: "windows", GOARCH: "amd64"}},
		{"windows-386", &Target{GOOS: "windows", GOARCH: "386"}},
		{"windows-arm64", &Target{GOOS: "windows", GOARCH: "arm64"}},
		{"wasm32", &Target{GOOS: "wasip1", GOARCH: "wasm"}},
		{"wasm64", &Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", WasmProvider: "emscripten"}},
	} {
		for _, llvmFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/LLVM-first=%v", target.name, llvmFirst), func(t *testing.T) {
				pkg := intrinsicDeclarationPackage(t, target.arch)
				var predeclared Function
				if !llvmFirst {
					predeclared = pkg.NewFunc("llvm.trap", NoArgsNoRet, CDecl)
				}
				buildIntrinsicTrapCaller(pkg, "first", nil)
				existing := pkg.Module().NamedFunction("llvm.trap")
				if existing.IsNil() || existing.IntrinsicID() == 0 {
					t.Fatal("LLVM did not declare the intrinsic")
				}
				before := existing.String()
				wrapped := pkg.NewFunc("llvm.trap", NoArgsNoRet, CDecl)
				buildIntrinsicTrapCaller(pkg, "second", wrapped)
				// Calling the wrapper is essential: an unused renamed intrinsic
				// declaration can escape LLVM's verifier diagnostic.
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatalf("invalid intrinsic call: %v\n%s", err, pkg.String())
				}
				if wrapped.impl.C != existing.C || pkg.FuncOf("llvm.trap") != wrapped {
					t.Fatal("package did not adopt the exact intrinsic declaration")
				}
				if predeclared != nil && wrapped != predeclared {
					t.Fatal("repeat declaration replaced the package function")
				}
				if llvmFirst && before != existing.String() {
					t.Fatal("adoption changed LLVM's intrinsic declaration attributes")
				}
				if !pkg.Module().NamedFunction("llvm.trap.1").IsNil() {
					t.Fatal("package created a renamed intrinsic")
				}
			})
		}
	}
}

func TestLLVMIntrinsicDeclarationConflicts(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, kind := range []string{"parameter", "result", "Go ABI", "stdcall ABI", "LLVM convention", "instantiated", "environment"} {
			t.Run(fmt.Sprintf("%s/cached=%v", kind, cached), func(t *testing.T) {
				var target *Target
				if kind == "stdcall ABI" {
					target = &Target{GOOS: "windows", GOARCH: "386"}
				}
				pkg := intrinsicDeclarationPackage(t, target)
				buildIntrinsicTrapCaller(pkg, "first", nil)
				if cached {
					pkg.NewFunc("llvm.trap", NoArgsNoRet, CDecl)
				}
				existing := pkg.Module().NamedFunction("llvm.trap")
				invoke := func() { pkg.NewFunc("llvm.trap", NoArgsNoRet, CDecl) }
				wantConflict := "calling convention"
				switch kind {
				case "parameter", "result":
					wantConflict = "function type"
					values := types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.Uint32]))
					var params, results *types.Tuple
					if kind == "parameter" {
						params = values
					} else {
						results = values
					}
					sig := types.NewSignatureType(nil, nil, nil, params, results, false)
					invoke = func() { pkg.NewFunc("llvm.trap", sig, CDecl) }
				case "Go ABI":
					invoke = func() { pkg.NewFunc("llvm.trap", NoArgsNoRet, GoDecl) }
				case "stdcall ABI":
					invoke = func() { pkg.NewFunc("llvm.trap", NoArgsNoRet, Stdcall) }
				case "LLVM convention":
					existing.SetFunctionCallConv(llvm.FastCallConv)
				case "instantiated":
					wantConflict = "instantiation"
					invoke = func() { pkg.NewFuncEx("llvm.trap", NoArgsNoRet, CDecl, false, true) }
				case "environment":
					wantConflict = "environment"
					env := types.NewParam(0, nil, "env", types.Typ[types.Uintptr])
					invoke = func() { pkg.NewEnvFunc("llvm.trap", NoArgsNoRet, CDecl, env, false) }
				}
				func() {
					defer func() {
						if got := fmt.Sprint(recover()); !strings.Contains(got, "conflicting") || !strings.Contains(got, wantConflict) {
							t.Errorf("incompatible intrinsic declaration panic = %q, want conflict about %s", got, wantConflict)
						}
					}()
					invoke()
				}()
				if kind == "LLVM convention" {
					existing.SetFunctionCallConv(llvm.CCallConv)
				}
				if !pkg.Module().NamedFunction("llvm.trap.1").IsNil() {
					t.Fatal("rejected declaration created a renamed intrinsic")
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if valid := pkg.NewFunc("llvm.trap", NoArgsNoRet, CDecl); valid.impl.C != existing.C {
					t.Fatal("rejection poisoned subsequent intrinsic lookup")
				}
			})
		}
	}
}

func TestLLVMOverloadedIntrinsicDeclaration(t *testing.T) {
	pkg := intrinsicDeclarationPackage(t, nil)
	value := types.NewParam(0, nil, "value", types.Typ[types.Uint32])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(value), types.NewTuple(value), false)
	first := pkg.NewFunc("first", sig, CDecl)
	b := first.MakeBody(1)
	swapped := b.impl.CreateIntrinsic(pkg.Prog.Uint32().ll, llvm.LookupIntrinsicID("llvm.bswap"), []llvm.Value{first.Param(0).impl}, "")
	b.Return(Expr{swapped, pkg.Prog.Uint32()})
	b.EndBuild()
	b.Dispose()
	existing := pkg.Module().NamedFunction("llvm.bswap.i32")
	before := existing.String()
	function := pkg.NewFunc("llvm.bswap.i32", sig, CDecl)
	second := pkg.NewFunc("second", sig, CDecl)
	b = second.MakeBody(1)
	b.Return(b.Call(function.Expr, second.Param(0)))
	b.EndBuild()
	b.Dispose()
	if function.impl.C != existing.C || before != existing.String() {
		t.Fatal("overloaded intrinsic declaration was replaced or modified")
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
