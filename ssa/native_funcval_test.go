package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNativeIntegerFuncvalAdapters(t *testing.T) {
	for _, test := range nativeIntegerABITests {
		for _, source := range []string{"declaration", "pointer", "named-Go-function", "nil", "Go-declaration"} {
			t.Run(test.target.GOOS+"/"+test.target.GOARCH+"/"+test.target.LLVMTarget+"/"+source, func(t *testing.T) {
				prog := NewProgram(&test.target)
				defer prog.Dispose()
				pkg := prog.NewPackage("p", "example.com/p")
				rawPkg := types.NewPackage(pkg.Path(), "p")
				params := types.NewTuple(
					types.NewVar(token.NoPos, nil, "signed", types.Typ[types.Int8]),
					types.NewVar(token.NoPos, nil, "unsigned8", types.Typ[types.Uint8]),
					types.NewVar(token.NoPos, nil, "signed16", types.Typ[types.Int16]),
					types.NewVar(token.NoPos, nil, "unsigned", types.Typ[types.Uint16]),
					types.NewVar(token.NoPos, nil, "flag", types.Typ[types.Bool]),
					types.NewVar(token.NoPos, nil, "signed32", types.Typ[types.Int32]),
					types.NewVar(token.NoPos, nil, "unsigned32", types.Typ[types.Uint32]),
				)
				resultType := types.Typ[types.Int32]
				if test.target.GOARCH == "wasm" {
					// Hosted Wasm widens native uintptr results to the Go word size.
					resultType = types.Typ[types.Uintptr]
				}
				results := types.NewTuple(types.NewVar(token.NoPos, nil, "", resultType))
				sig := types.NewSignatureType(nil, nil, nil, params, results, false)
				pointer := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "Native", nil), sig, nil)
				prog.SetTypeCallConv(pkg.Path()+".Native", CDecl)
				ownerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
					types.NewVar(token.NoPos, nil, "fn", pointer),
				), nil, false)
				owner := pkg.NewFunc(pkg.Path()+".owner", ownerSig, GoDecl)
				body := owner.MakeBody(1)
				dst := prog.Type(sig, GoDecl)
				value := owner.Param(0)
				switch source {
				case "declaration":
					value = pkg.NewFunc("native", sig, CDecl).Expr
				case "named-Go-function":
					named := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "GoFunction", nil), sig, nil)
					dst = prog.Type(named, GoDecl)
				case "nil":
					value = prog.Nil(prog.Type(pointer, GoDecl))
				case "Go-declaration":
					value = pkg.NewFunc(pkg.Path()+".goFunction", sig, GoDecl).Expr
				}
				var converted Expr
				if source == "declaration" || source == "pointer" {
					// Assignment/argument lowering and explicit conversion use
					// separate entry points; both must adapt the C boundary.
					converted = checkExpr(value, dst.raw.Type, body)
				} else {
					converted = body.ChangeType(dst, value)
				}
				if converted.kind != vkClosure || converted.impl.Type() != dst.ll {
					t.Fatalf("conversion did not produce a Go function value: %s", converted.impl)
				}
				if source == "nil" && (!body.Field(converted, 0).impl.IsNull() || !body.Field(converted, 1).impl.IsNull()) {
					t.Fatalf("nil native pointer produced non-nil Go function: %s", converted.impl)
				}
				body.Return()
				goSig := dst.raw.Type.Underlying().(*types.Struct).Field(0).Type().(*types.Signature)
				wrapper := pkg.FuncOf(pkg.Path() + ".__llgo_c_funcval$" + prog.abi.FuncName(goSig))
				wantAdapter := test.attrs != nil && source != "Go-declaration"
				if (wrapper != nil) != wantAdapter {
					t.Fatalf("native adapter present = %v, want %v:\n%s", wrapper != nil, wantAdapter, pkg.String())
				}
				if wantAdapter {
					if !wrapper.NeedsEnv() || wrapper.impl.Linkage() != llvm.InternalLinkage {
						t.Fatalf("adapter must be an internal Go closure entry:\n%s", wrapper.impl)
					}
					if source != "nil" && !prog.usesWideGoStorage() && converted.impl.Operand(1) != value.impl {
						t.Fatalf("adapter environment did not retain native pointer: %s", converted.impl)
					}
					var nativeCall llvm.Value
					for block := wrapper.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
						for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
							if call := instruction.IsACallInst(); !call.IsNil() && call.CalledValue() == wrapper.Env().impl {
								nativeCall = call
							}
						}
					}
					if nativeCall.IsNil() {
						t.Fatalf("adapter does not call its native pointer:\n%s", wrapper.impl)
					}
					checkNarrowAttrs(t, nativeCall, true, test.attrs)
					checkNarrowAttrs(t, wrapper.impl, false, make([]string, len(test.attrs)+1))
					if test.target.WasmProfile == "w32" && !strings.Contains(wrapper.impl.String(), "zext i32") {
						t.Fatalf("adapter did not widen the native uintptr result:\n%s", wrapper.impl)
					}
					if strings.Contains(pkg.String(), "runtime.Alloc") {
						t.Fatalf("native function conversion unexpectedly allocated:\n%s", pkg.String())
					}
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatalf("invalid native function conversion: %v\n%s", err, pkg.String())
				}
			})
		}
	}
}

func TestNativeIntegerFuncvalRejectsVariadicForwarding(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "darwin", GOARCH: "arm64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
		types.NewVar(token.NoPos, nil, "fixed", types.Typ[types.Int8]), VArg(),
	), nil, true)
	native := pkg.NewFunc("native", sig, CDecl)
	body := pkg.NewFunc(pkg.Path()+".owner", NoArgsNoRet, GoDecl).MakeBody(1)
	defer func() {
		got := recover()
		if got == nil || !strings.Contains(fmt.Sprint(got), "C variadic functions require a native function pointer") {
			t.Fatalf("variadic conversion panic = %v, want native function pointer diagnostic", got)
		}
	}()
	body.ChangeType(prog.Type(sig, GoDecl), native.Expr)
}

func TestNativeFunctionPointerStore(t *testing.T) {
	for _, test := range nativeIntegerABITests {
		for _, source := range []string{"nil", "native-pointer", "Go-closure"} {
			t.Run(test.target.GOOS+"/"+test.target.GOARCH+"/"+test.target.LLVMTarget+"/"+source, func(t *testing.T) {
				prog := NewProgram(&test.target)
				defer prog.Dispose()
				setTestRuntime(t, prog)
				pkg := prog.NewPackage("p", "example.com/p")
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
					types.NewParam(token.NoPos, nil, "fn", NoArgsNoRet),
				), nil, false)
				owner := pkg.NewFunc(pkg.Path()+".store", sig, GoDecl)
				body := owner.MakeBody(1)
				global := pkg.NewVar("native_callback", types.NewPointer(NoArgsNoRet), CDecl)
				value := prog.Nil(prog.Type(NoArgsNoRet, GoDecl))
				switch source {
				case "native-pointer":
					value = body.Load(global.Expr)
				case "Go-closure":
					value = body.Param(0)
				}
				stored := body.Store(global.Expr, value)
				if got := stored.impl.Operand(0).Type(); got.TypeKind() != llvm.PointerTypeKind {
					t.Fatalf("native function pointer store writes %s instead of one pointer:\n%s", got, stored.impl)
				}
				body.Return()
				body.EndBuild()
				if source == "Go-closure" && !strings.Contains(pkg.String(), "C callback must not capture variables") {
					t.Fatalf("native callback assignment lost the closure environment check:\n%s", pkg.String())
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatalf("invalid native function pointer store: %v\n%s", err, pkg.String())
				}
			})
		}
	}
}
