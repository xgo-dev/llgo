package build

import (
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestFilterABIFunctionValueTypeFamily(t *testing.T) {
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	closure := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "$f", sig, false),
		types.NewField(token.NoPos, nil, "$data", types.Typ[types.UnsafePointer], false),
	}, nil)
	ordinary := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Code", sig, false),
		types.NewField(token.NoPos, nil, "Data", types.Typ[types.UnsafePointer], false),
	}, nil)
	if !llssa.IsClosure(closure) || llssa.IsClosure(ordinary) {
		t.Fatal("typed function/ordinary struct fixtures are not distinct")
	}
	for _, typ := range []struct {
		name     string
		raw      types.Type
		function bool
	}{
		{"Go-function-value", closure, true},
		{"native-function-signature", sig, true},
		{"ordinary-struct", ordinary, false},
	} {
		for _, flag := range []struct {
			name string
			mask int
		}{
			{"none", 0},
			{"array", llssa.ReflectArrayOf},
			{"channel", llssa.ReflectChanOf},
			{"function", llssa.ReflectFuncOf},
			{"map", llssa.ReflectMapOf},
			{"pointer", llssa.ReflectPointerTo},
			{"slice", llssa.ReflectSliceOf},
			{"struct", llssa.ReflectStructOf},
			{"method-index", llssa.ReflectMethodByIndex},
			{"method-name", llssa.ReflectMethodByName},
			{"method-dynamic", llssa.ReflectMethodDynamic},
			{"type-method-index", llssa.ReflectTypeMethodByIndex},
			{"type-method-name", llssa.ReflectTypeMethodByName},
			{"type-method-dynamic", llssa.ReflectTypeMethodDynamic},
		} {
			t.Run(typ.name+"/"+flag.name, func(t *testing.T) {
				want := flag.mask&llssa.ReflectStructOf != 0
				if typ.function {
					want = flag.mask&(llssa.ReflectFuncOf|llssa.ReflectMethodMask) != 0
				}
				if got := filterAbiSymbol(flag.mask, &llssa.AbiSymbol{Raw: typ.raw}); got != want {
					t.Fatalf("filter = %v, want %v for reflection mask %#x", got, want, flag.mask)
				}
			})
		}
	}
}
