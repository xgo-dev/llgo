package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// The public builder must emit verifier-valid IR across the ABI and epilogue
// boundaries where a native adapter cannot guarantee a forwarding tail call.
func TestRecoverForwardTailABI(t *testing.T) {
	for _, name := range []string{"scalar", "scoped", "no alias", "nonrecovering", "amd64", "roots", "closure context", "disabled", "different prototype", "different convention", "aggregate", "converted result", "native narrow argument"} {
		t.Run(name, func(t *testing.T) {
			arch := "386"
			if name == "amd64" {
				arch = "amd64"
			}
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			setTestRuntime(t, prog)
			prog.EnableGCRoots(name == "roots")
			pkg := prog.NewPackage("p", "example.com/forward")
			params := []*types.Var{
				types.NewVar(token.NoPos, nil, "pointer", types.Typ[types.UnsafePointer]),
				types.NewVar(token.NoPos, nil, "integer", types.Typ[types.Int32]),
				types.NewVar(token.NoPos, nil, "float", types.Typ[types.Float32]),
				types.NewVar(token.NoPos, nil, "double", types.Typ[types.Float64]),
			}
			if name == "aggregate" {
				params[0] = types.NewVar(token.NoPos, nil, "record", types.NewStruct([]*types.Var{types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32])}, nil))
			}
			if name == "native narrow argument" {
				params[1] = types.NewVar(token.NoPos, nil, "narrow", types.Typ[types.Int8])
			}
			results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64]))
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), results, false)
			calleeSig := sig
			if name == "different prototype" {
				calleeSig = types.NewSignatureType(nil, nil, nil, types.NewTuple(params[:3]...), results, false)
			}
			cc := GoDecl
			if name == "different convention" {
				cc = Stdcall
			}
			if name == "native narrow argument" {
				cc = CDecl
			}
			callee := pkg.NewFunc("callee", calleeSig, cc)
			var caller Function
			if name == "closure context" {
				caller = pkg.NewEnvFunc("caller", sig, GoDecl, types.NewVar(token.NoPos, nil, "env", types.Typ[types.UnsafePointer]), false)
			} else {
				caller = pkg.NewFunc("caller", sig, GoDecl)
			}
			if name == "disabled" {
				caller.DisableTailCalls()
			}
			body := caller.MakeBody(1)
			args := make([]Expr, calleeSig.Params().Len())
			for i := range args {
				args[i] = caller.Param(i)
			}
			from := caller.Expr
			if name == "no alias" {
				from = Nil
			}
			buildCall := Builder.Call
			if name == "converted result" {
				buildCall = func(b Builder, fn Expr, args ...Expr) Expr {
					return b.BinOp(token.ADD, b.Call(fn, args...), prog.IntVal(1, prog.Int64()))
				}
			}
			var result Expr
			if name == "scoped" {
				result = body.CallRecoverAlias(from, true, callee.Expr, buildCall, args...)
			} else {
				result = body.CallRecoverTailAlias(from, name != "nonrecovering", callee.Expr, buildCall, args...)
			}
			body.Return(result)
			body.EndBuild()
			ir := caller.impl.String()
			wantTail := name == "scalar"
			if strings.Contains(ir, "musttail") != wantTail {
				t.Fatalf("musttail=%v, want %v:\n%s", strings.Contains(ir, "musttail"), wantTail, ir)
			}
			if strings.Contains(ir, "EndRecoverFrameAlias") != (name == "scoped") {
				t.Fatalf("incorrect recover cleanup:\n%s", ir)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid adapter ABI: %v\n%s", err, ir)
			}
		})
	}
}
