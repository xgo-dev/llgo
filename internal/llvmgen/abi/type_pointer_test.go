package abi

import (
	"fmt"
	"go/types"
	"strings"
	"testing"
)

func TestHasPointers(t *testing.T) {
	integer := types.Typ[types.Int]
	pointer := types.NewPointer(integer)
	fields := func(ts ...types.Type) *types.Struct {
		vars := make([]*types.Var, len(ts))
		for i, typ := range ts {
			vars[i] = types.NewField(0, nil, fmt.Sprintf("F%d", i), typ, false)
		}
		return types.NewStruct(vars, nil)
	}
	named := func(name string, typ types.Type) *types.Named {
		return types.NewNamed(types.NewTypeName(0, nil, name, nil), typ, nil)
	}
	for _, arch := range []string{"amd64", "386"} {
		t.Run(arch, func(t *testing.T) {
			sizes := types.SizesFor("gc", arch)
			b := New(uintptr(sizes.Sizeof(types.Typ[types.Uintptr])), sizes)
			for _, tc := range []struct {
				name string
				typ  types.Type
				want bool
			}{
				{"integer", integer, false},
				{"uintptr", types.Typ[types.Uintptr], false},
				{"string", types.Typ[types.String], true},
				{"unsafePointer", types.Typ[types.UnsafePointer], true},
				{"pointer", pointer, true},
				{"slice", types.NewSlice(integer), true},
				{"function", types.NewSignatureType(nil, nil, nil, nil, nil, false), true},
				{"interface", types.NewInterfaceType(nil, nil).Complete(), true},
				{"map", types.NewMap(integer, integer), true},
				{"channel", types.NewChan(types.SendRecv, integer), true},
				{"emptyStruct", fields(), false},
				{"scalarStruct", fields(integer), false},
				{"pointerStruct", fields(integer, pointer), true},
				{"nestedStruct", fields(integer, fields(pointer)), true},
				{"scalarArray", types.NewArray(integer, 3), false},
				{"pointerArray", types.NewArray(pointer, 3), true},
				{"emptyPointerArray", types.NewArray(pointer, 0), false},
				{"namedScalar", named("Scalar", integer), false},
				{"namedPointer", named("Pointer", pointer), true},
				{"aliasScalar", types.NewAlias(types.NewTypeName(0, nil, "ScalarAlias", nil), integer), false},
				{"aliasPointer", types.NewAlias(types.NewTypeName(0, nil, "PointerAlias", nil), pointer), true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if got := b.HasPointers(tc.typ); got != tc.want {
						t.Fatalf("HasPointers(%s) = %v, want %v", tc.typ, got, tc.want)
					}
					if got := b.PtrBytes(tc.typ) != 0; got != tc.want {
						t.Fatalf("PtrBytes(%s) != 0 is %v, want %v", tc.typ, got, tc.want)
					}
				})
			}
			t.Run("unsupportedTuple", func(t *testing.T) {
				defer func() {
					if got, ok := recover().(string); !ok || !strings.Contains(got, "unsupported pointer layout") {
						t.Fatalf("HasPointers(tuple) panic = %v, want unsupported pointer layout", got)
					}
				}()
				b.HasPointers(types.NewTuple(types.NewVar(0, nil, "", integer)))
			})
		})
	}
}
