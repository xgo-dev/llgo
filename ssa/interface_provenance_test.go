package ssa

import (
	"go/token"
	"go/types"
	"runtime"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestInterfaceConcreteTypeProvenance(t *testing.T) {
	for _, name := range []string{"shared-nil-data", "shared-address", "replaced-type-slot", "mixed-select", "same-type-select", "shared-select", "unknown-parameter", "undef", "same-type-phi", "mixed-phi", "unknown-phi", "cyclic-phi", "bitcast", "empty-proof"} {
		t.Run(name, func(t *testing.T) {
			prog := NewProgram(nil)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
			pkg := prog.NewPackage("provenance", "example.com/provenance")
			face := prog.Struct(prog.VoidPtr(), prog.VoidPtr())
			params := types.NewTuple(
				types.NewVar(token.NoPos, nil, "left", face.RawType()),
				types.NewVar(token.NoPos, nil, "right", face.RawType()),
				types.NewVar(token.NoPos, nil, "condition", types.Typ[types.Bool]),
				types.NewVar(token.NoPos, nil, "address", types.Typ[types.UnsafePointer]),
			)
			fn := pkg.NewFunc("Check", types.NewSignatureType(nil, nil, nil, params, nil, false), GoDecl)
			b := fn.MakeBody(1)
			left, right := fn.Param(0).impl, fn.Param(1).impl
			a := types.NewPointer(types.Typ[types.Int])
			other := types.NewPointer(types.Typ[types.String])
			b.Func.interfaceConcreteTypes = map[llvm.Value]types.Type{left: a, right: other}
			var value llvm.Value
			var want types.Type
			switch name {
			case "shared-nil-data", "shared-address":
				data := prog.Zero(prog.VoidPtr()).impl
				if name == "shared-address" {
					data = fn.Param(3).impl
				}
				// The same data word may occur in interfaces with different types.
				// A later construction must not determine an earlier value's type.
				value = b.impl.CreateInsertValue(left, data, 1, "")
				_ = b.impl.CreateInsertValue(right, data, 1, "")
				want = a
			case "replaced-type-slot":
				// Replacing the type word invalidates the original type proof.
				value = b.impl.CreateInsertValue(left, b.impl.CreateExtractValue(right, 0, ""), 0, "")
			case "empty-proof":
				b.Func.interfaceConcreteTypes = nil
				value = left
			case "bitcast":
				bits := b.impl.CreatePtrToInt(fn.Param(3).impl, prog.Int64().ll, "")
				b.Func.interfaceConcreteTypes[bits] = a
				value = b.impl.CreateBitCast(bits, prog.Float64().ll, "")
				want = a
			case "same-type-phi", "mixed-phi", "unknown-phi", "cyclic-phi":
				entry := b.impl.GetInsertBlock()
				merge := prog.ctx.AddBasicBlock(fn.impl, "merge")
				if name == "cyclic-phi" {
					b.impl.CreateBr(merge)
					b.impl.SetInsertPointAtEnd(merge)
					value = b.impl.CreatePHI(left.Type(), "cycle")
					value.AddIncoming([]llvm.Value{left, value}, []llvm.BasicBlock{entry, merge})
					exit := prog.ctx.AddBasicBlock(fn.impl, "exit")
					b.impl.CreateCondBr(fn.Param(2).impl, merge, exit)
					b.impl.SetInsertPointAtEnd(exit)
				} else {
					alternate := prog.ctx.AddBasicBlock(fn.impl, "alternate")
					b.impl.CreateCondBr(fn.Param(2).impl, merge, alternate)
					b.impl.SetInsertPointAtEnd(alternate)
					b.impl.CreateBr(merge)
					b.impl.SetInsertPointAtEnd(merge)
					value = b.impl.CreatePHI(left.Type(), "merged")
					if name == "same-type-phi" {
						b.Func.interfaceConcreteTypes[right] = a
						want = a
					} else if name == "unknown-phi" {
						delete(b.Func.interfaceConcreteTypes, right)
					}
					value.AddIncoming([]llvm.Value{left, right}, []llvm.BasicBlock{entry, alternate})
				}
			case "mixed-select":
				value = b.impl.CreateSelect(fn.Param(2).impl, left, right, "")
			case "unknown-parameter":
				value = fn.Param(3).impl
			case "undef":
				value = llvm.Undef(left.Type())
			case "shared-select":
				b.Func.interfaceConcreteTypes[right] = a
				inner := b.impl.CreateSelect(fn.Param(2).impl, left, right, "")
				value = b.impl.CreateSelect(fn.Param(2).impl, inner, left, "")
				want = a
			case "same-type-select":
				b.Func.interfaceConcreteTypes[right] = a
				value = b.impl.CreateSelect(fn.Param(2).impl, left, right, "")
				want = a
			}
			got := b.knownInterfaceConcreteType(value)
			// Conservative loss of a known type is safe; a conflicting proof is not.
			if got != nil && (want == nil || !types.Identical(got, want)) {
				t.Errorf("inferred concrete type = %v, want %v or unknown", got, want)
			}
			if (name == "same-type-select" || name == "shared-select" || name == "same-type-phi" || name == "bitcast") && got == nil {
				t.Error("value-preserving merge or cast lost its concrete type proof")
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
