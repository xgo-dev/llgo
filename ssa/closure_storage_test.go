package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestMakeClosureExCapturedStorage(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
	} {
		for _, mode := range []string{"default-heap", "explicit-heap", "stack"} {
			t.Run(target.GOOS+"/"+target.GOARCH+"/"+mode, func(t *testing.T) {
				prog := NewProgram(target)
				defer prog.Dispose()
				prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
				setTestRuntime(t, prog)
				pkg := prog.NewPackage("storage", "example.com/storage")
				result := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
				sig := types.NewSignatureType(nil, nil, nil, nil, result, false)
				captures := types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "n", types.Typ[types.Int], false)}, nil)
				env := types.NewParam(token.NoPos, nil, "$env", types.NewPointer(captures))
				entry := pkg.NewEnvFunc("entry", sig, GoDecl, env, false)
				inner := entry.MakeBody(1)
				inner.Return(inner.Load(inner.FieldAddr(entry.Env(), 0)))
				inner.EndBuild()
				params := types.NewTuple(types.NewParam(token.NoPos, nil, "n", types.Typ[types.Int]))
				caller := pkg.NewFunc("caller", types.NewSignatureType(nil, nil, nil, params, result, false), GoDecl)
				body := caller.MakeBody(1)
				bindings := []Expr{caller.Param(0)}
				var value Expr
				if mode == "default-heap" {
					value = body.MakeClosure(entry.Expr, bindings)
				} else {
					value = body.MakeClosureEx(entry.Expr, bindings, mode != "stack")
				}
				body.Return(body.CallKnownClosure(value, entry))
				body.EndBuild()
				if value.ll.TypeKind() != llvm.PointerTypeKind || prog.SizeOf(value.Type) != uint64(prog.GoWordSize()) {
					t.Fatal("allocation choice changed the one-word function value")
				}
				want := 1
				if mode == "stack" {
					want = 0
				}
				assertFuncvalCollectibleAllocations(t, caller, want)
				allocation := value.impl.IsAGetElementPtrInst()
				if allocation.IsNil() {
					t.Fatal("captured value lost its header within the shared allocation", caller.impl)
				}
				allocation = allocation.Operand(0)
				if mode == "stack" && allocation.IsAAllocaInst().IsNil() {
					t.Fatal("non-escaping header/captures were not allocated together on the stack", caller.impl)
				}
				assertFuncvalHeaderPart(t, caller, value, 0, entry.impl)
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err, pkg.String())
				}
			})
		}
	}
}

func TestMakeClosureExZeroSizedEnvironment(t *testing.T) {
	for _, heap := range []bool{false, true} {
		name := "stack"
		if heap {
			name = "heap"
		}
		t.Run(name, func(t *testing.T) {
			prog := NewProgram(nil)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("zero", "example.com/zero")
			empty := types.NewStruct(nil, nil)
			env := types.NewParam(token.NoPos, nil, "$env", types.NewPointer(empty))
			entry := pkg.NewEnvFunc("entry", NoArgsNoRet, GoDecl, env, false)
			inner := entry.MakeBody(1)
			inner.Return()
			inner.EndBuild()
			caller := pkg.NewFunc("caller", NoArgsNoRet, GoDecl)
			body := caller.MakeBody(1)
			value := body.MakeClosureEx(entry.Expr, nil, heap)
			context := body.funcvalField(value, 1)
			if context.impl.IsNull() || value.impl.IsNull() {
				t.Fatal("required zero-sized environment became a no-env/nil function")
			}
			body.CallKnownClosure(value, entry)
			body.Return()
			body.EndBuild()
			assertFuncvalCollectibleAllocations(t, caller, 0)
			assertFuncvalHeaderPart(t, caller, value, 0, entry.impl)
			assertFuncvalHeaderPart(t, caller, value, 1, context.impl)
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err, pkg.String())
			}
		})
	}
}
