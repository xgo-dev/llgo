package cl

import (
	"go/token"
	"go/types"
	"testing"
)

func TestNoEscapeInstructionSignature(t *testing.T) {
	pointer := types.Typ[types.UnsafePointer]
	integer := types.Typ[types.Int]
	for _, test := range []struct {
		name           string
		params, result []types.Type
		want           bool
	}{
		{"pointer", []types.Type{pointer}, []types.Type{pointer}, true},
		{"integer", []types.Type{integer}, []types.Type{integer}, false},
		{"missing input", nil, []types.Type{pointer}, false},
		{"missing result", []types.Type{pointer}, nil, false},
		{"extra input", []types.Type{pointer, pointer}, []types.Type{pointer}, false},
		{"extra result", []types.Type{pointer}, []types.Type{pointer, pointer}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			tuple := func(ts []types.Type) *types.Tuple {
				vars := make([]*types.Var, len(ts))
				for i, typ := range ts {
					vars[i] = types.NewVar(token.NoPos, nil, "", typ)
				}
				return types.NewTuple(vars...)
			}
			sig := types.NewSignatureType(nil, nil, nil, tuple(test.params), tuple(test.result), false)
			if _, ok := instructionEscapeContract(llgoNoEscape, sig); ok != test.want {
				t.Fatalf("contract accepted=%v, want %v", ok, test.want)
			}
		})
	}
}
