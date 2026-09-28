package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

func TestPointerAttributesAcrossBackendsAndInstrumentation(t *testing.T) {
	for _, mode := range []string{"none", "roots", "safepoints"} {
		t.Run(mode, func(t *testing.T) {
			coordinator := NewProgram(nil)
			defer coordinator.Dispose()
			coordinator.EnableGCRoots(mode == "roots")
			coordinator.EnableCooperativeSafepoints(mode == "safepoints")
			var attrs []funcattrs.Attribute
			for _, name := range []string{"noalias", "access", "nonnull"} {
				attr := funcattrs.Attribute{Target: funcattrs.Target{Scope: funcattrs.Parameter}, Name: name, Position: token.Position{Filename: "pointers.go", Line: 3}}
				if name == "access" {
					attr.Args = "read"
				}
				attrs = append(attrs, attr)
			}

			ptr := types.NewPointer(types.Typ[types.Int])
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "p", ptr)), nil, false)
			for _, owner := range []string{"p", "caller"} {
				backend := coordinator.NewBackendProgram()
				defer backend.Dispose()
				pkg := backend.NewPackage(owner, owner)
				env := types.NewParam(token.NoPos, nil, "$env", types.Typ[types.UnsafePointer])
				fn := pkg.NewEnvFunc("p.F[int]", sig, InGo, env, false)
				fn.ApplyValueAttributes(sig, attrs)
				if owner == "p" {
					b := fn.MakeBody(1)
					b.Return()
					b.EndBuild()
				}
				for _, name := range []string{"noalias", "readonly", "nonnull"} {
					kind := llvm.AttributeKindID(name)
					if !fn.impl.GetEnumAttributeAtIndex(1, kind).IsNil() {
						t.Fatal("attribute on hidden environment")
					}
					if !fn.impl.GetEnumAttributeAtIndex(2, kind).IsNil() != (mode == "none" || name == "nonnull") {
						t.Fatalf("%s: %s policy for %s", owner, mode, name)
					}
				}
				err := backend.CheckPointerEffects(fn.impl, "test instrumentation")
				if mode == "none" {
					if err == nil || !strings.Contains(err.Error(), "pointers.go:3") {
						t.Fatal(err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				if err = llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPointerAttributesRejectLateABIAllocation(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	attr := funcattrs.Attribute{Target: funcattrs.Target{Scope: funcattrs.Parameter}, Name: "noalias", Position: token.Position{Filename: "late.go", Line: 4}}

	ptr := types.NewPointer(types.Typ[types.Int])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "p", ptr)), nil, false)
	pkg := prog.NewPackage("p", "p")
	fn := pkg.NewFunc("p.F", sig, InGo)
	fn.ApplyValueAttributes(sig, []funcattrs.Attribute{attr})
	mod := pkg.Module()
	ctx := mod.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn.impl, "entry"))
	calleeType := llvm.FunctionType(llvm.ArrayType(ctx.Int8Type(), 80000), nil, false)
	callee := llvm.AddFunction(mod, "large", calleeType)
	llvm.CreateCall(b, calleeType, callee, nil)
	b.CreateRetVoid()
	defer func() {
		failure := recover()
		err, ok := failure.(error)
		if !ok || !strings.Contains(err.Error(), "late.go:4") || !strings.Contains(err.Error(), "large ABI result allocation") {
			t.Fatalf("unexpected diagnostic: %v", failure)
		}
	}()
	abi.LowerLargeAggregates(prog.TargetData(), mod, abi.AggregateLoweringConfig{CheckEffects: prog.CheckPointerEffects})
}

func TestPointerEffectsRetainAllKnownRestrictions(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	ptr := types.NewPointer(types.Typ[types.Int])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(0, nil, "p", ptr)), nil, false)
	fn := prog.NewPackage("p", "p").NewFunc("shared", sig, InGo)
	fn.ApplyValueAttributes(sig, []funcattrs.Attribute{{Target: funcattrs.Target{Scope: funcattrs.Parameter}, Name: "noalias", Position: token.Position{Filename: "alias.go", Line: 2}}})
	fn.ApplyValueAttributes(sig, []funcattrs.Attribute{{Target: funcattrs.Target{Scope: funcattrs.Parameter}, Name: "access", Args: "readwrite"}})
	if err := prog.CheckPointerEffects(fn.impl, "late allocation"); err == nil || !strings.Contains(err.Error(), "alias.go:2") {
		t.Fatalf("lost earlier retained restriction: %v", err)
	}
}
