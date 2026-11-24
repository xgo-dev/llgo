package ssa

import (
	"go/token"
	"go/types"
	"runtime"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestInterfaceConcreteTypeProvenance(t *testing.T) {
	for _, name := range []string{"shared-nil-data", "shared-address", "replaced-type-slot", "mixed-select", "same-type-select", "shared-select", "unknown-parameter", "undef"} {
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
			if (name == "same-type-select" || name == "shared-select") && got == nil {
				t.Error("same-type select lost its concrete type proof")
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
