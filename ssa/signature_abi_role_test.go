package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func variadicRoleSignature(ellipsis bool) *types.Signature {
	name := "values"
	if ellipsis {
		name = "__llgo_va_list"
	}
	params := types.NewTuple(
		types.NewVar(token.NoPos, nil, "marker", types.Typ[types.Int32]),
		types.NewVar(token.NoPos, nil, name, types.NewSlice(types.NewInterfaceType(nil, nil).Complete())),
	)
	result := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]))
	return types.NewSignatureType(nil, nil, nil, params, result, true)
}

func TestFunctionSignatureCacheRetainsVariadicABI(t *testing.T) {
	for _, target := range []Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"}} {
		for _, first := range []bool{false, true} {
			order := "fixed-first"
			if first {
				order = "ellipsis-first"
			}
			t.Run(target.GOOS+"/"+target.GOARCH+"/"+order, func(t *testing.T) {
				prog := NewProgram(&target)
				defer prog.Dispose()
				setTestRuntime(t, prog)
				prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
				fixed, ellipsis := variadicRoleSignature(false), variadicRoleSignature(true)
				if !types.Identical(fixed, ellipsis) {
					t.Fatal("fixture must have Go-identical signatures with different physical ABIs")
				}
				signatures := map[bool]*types.Signature{false: fixed, true: ellipsis}
				initial := prog.Type(signatures[first], CDecl)
				if again := prog.Type(signatures[first], CDecl); again != initial {
					t.Fatal("ordinary same-role cache hits must retain the existing Type")
				}

				// Native pointers must not replace the source signature stored in
				// a Go descriptor header, in either construction order.
				closure := prog.Closure(fixed)
				code := prog.Field(closure, 0).RawType().(*types.Signature)
				header := prog.funcvalDescriptorType(closure)
				headerCode := header.RawType().(*types.Struct).Field(0).Type().(*types.Signature)
				if HasNameValist(code) || HasNameValist(headerCode) {
					t.Error("ordinary Go function/header acquired the native ellipsis role")
				}
				if prog.funcvalDescriptorType(closure) != header {
					t.Error("same-role descriptor header lost its existing cache")
				}

				pkg := prog.NewPackage("roles", "example.com/roles")
				for _, want := range []bool{first, !first} {
					name := "fixed"
					if want {
						name = "ellipsis"
					}
					t.Run(name, func(t *testing.T) {
						signature := signatures[want]
						pointer := prog.Type(signature, CDecl)
						if got := HasNameValist(pointer.RawType().(*types.Signature)); got != want {
							t.Fatalf("native pointer variadic ABI = %v, want %v", got, want)
						}
						params := types.NewTuple(types.NewVar(token.NoPos, nil, "callback", signature))
						caller := pkg.NewFunc("roles."+name, types.NewSignatureType(nil, nil, nil, params, signature.Results(), false), CDecl)
						body := caller.MakeBody(1)
						args := []Expr{prog.IntVal(7, prog.Int32())}
						if want {
							args = append(args, prog.IntVal(19, prog.Int32()))
						} else {
							args = append(args, prog.Nil(prog.Type(signature.Params().At(1).Type(), CDecl)))
						}
						// Bind the physical native pointer view explicitly. The CL
						// source owner separately verifies annotated parameter lowering.
						callback := Expr{caller.impl.Param(0), pointer}
						result := body.Call(callback, args...)
						body.Return(result)
						callType := result.impl.CalledFunctionType()
						count := 2
						if want {
							count = 1
						}
						if callType.IsFunctionVarArg() != want || callType.ParamTypesCount() != count {
							t.Fatalf("native callback has the wrong physical prototype: %s", callType)
						}
					})
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestNativeFunctionConversionRejectsVariadicABIMismatch(t *testing.T) {
	for _, fromEllipsis := range []bool{false, true} {
		for _, mismatch := range []bool{false, true} {
			fromName, toName := "fixed", "fixed"
			if fromEllipsis {
				fromName, toName = "ellipsis", "ellipsis"
			}
			if mismatch {
				toName = "ellipsis"
				if fromEllipsis {
					toName = "fixed"
				}
			}
			t.Run(fromName+"-to-"+toName, func(t *testing.T) {
				prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
				defer prog.Dispose()
				setTestRuntime(t, prog)
				pkg := prog.NewPackage("roles", "example.com/roles")
				from, to := variadicRoleSignature(fromEllipsis), variadicRoleSignature(fromEllipsis != mismatch)
				// Independent source-preserving views isolate conversion from the
				// separately tested shared-cache bug; no runtime address is guessed.
				source, destination := prog.toType(from), prog.toType(to)
				entry := pkg.NewFunc("roles.entry", from, CDecl)
				caller := pkg.NewFunc("roles.convert", NoArgsNoRet, GoDecl)
				body := caller.MakeBody(1)
				defer func() {
					value := recover()
					if !mismatch {
						if value != nil {
							t.Errorf("same-role native conversion was rejected: %v", value)
						}
					} else if value == nil {
						t.Error("conversion silently changed the native fixed-slice/ellipsis ABI")
					} else if !strings.Contains(fmt.Sprint(value), "variadic") {
						t.Errorf("unexpected conversion rejection: %v", value)
					}
				}()
				converted := body.ChangeType(destination, Expr{entry.impl, source})
				if !mismatch && (converted.impl != entry.impl || HasNameValist(converted.RawType().(*types.Signature)) != fromEllipsis) {
					t.Error("same-role conversion changed the function pointer or prototype role")
				}
				body.Return()
			})
		}
	}
}
