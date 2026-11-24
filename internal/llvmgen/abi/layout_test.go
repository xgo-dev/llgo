package abi

import (
	"go/token"
	"go/types"
	"testing"
)

func TestAlignUsesTargetGoSizes(t *testing.T) {
	for _, test := range []struct {
		arch string
		want uintptr
	}{
		{arch: "386", want: 4},
		{arch: "amd64", want: 8},
		{arch: "arm64", want: 8},
	} {
		builder := New(test.want, types.SizesFor("gc", test.arch))
		if got := builder.Align(types.Typ[types.Uint64]); got != test.want {
			t.Errorf("Align(uint64) on %s = %d, want %d", test.arch, got, test.want)
		}
		st := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "B", types.Typ[types.Byte], false),
			types.NewField(token.NoPos, nil, "U", types.Typ[types.Uint64], false),
		}, nil)
		if got := builder.Align(st); got != test.want {
			t.Errorf("Align(struct{byte; uint64}) on %s = %d, want %d", test.arch, got, test.want)
		}
	}
}
