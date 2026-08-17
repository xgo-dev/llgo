//go:build !llgo

package ssa_test

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

func TestBoundsCheckModesIR(t *testing.T) {
	checked := boundsCheckModeIR(t, false)
	unchecked := boundsCheckModeIR(t, true)

	for _, helper := range []string{
		"CheckIndexRange",
		"StringSlice2",
		"NewSlice2",
		"NewSlice3Bounds",
		"PanicSliceConvert",
	} {
		if !strings.Contains(checked, helper) {
			t.Errorf("checked IR does not contain %q", helper)
		}
		if strings.Contains(unchecked, helper) {
			t.Errorf("unchecked IR contains %q", helper)
		}
	}
	if !strings.Contains(unchecked, "AssertNilDeref") {
		t.Error("unchecked *array slice lost its nil check")
	}
	if got := strings.Count(unchecked, "select i1"); got < 4 {
		t.Errorf("unchecked IR contains %d select operations, want at least 4", got)
	}
}

func boundsCheckModeIR(t *testing.T, disable bool) string {
	t.Helper()
	prog := ssatest.NewProgram(t, nil)
	t.Cleanup(prog.Dispose)
	prog.DisableBoundsChecks(disable)

	byteSlice := types.NewSlice(types.Typ[types.Byte])
	byteArray := types.NewArray(types.Typ[types.Byte], 4)
	byteArrayPtr := types.NewPointer(byteArray)
	params := types.NewTuple(
		types.NewVar(0, nil, "str", types.Typ[types.String]),
		types.NewVar(0, nil, "slice", byteSlice),
		types.NewVar(0, nil, "array", byteArrayPtr),
		types.NewVar(0, nil, "low", types.Typ[types.Int]),
		types.NewVar(0, nil, "high", types.Typ[types.Int]),
		types.NewVar(0, nil, "max", types.Typ[types.Int]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	pkg := prog.NewPackage("bounds", "example.com/bounds")
	fn := pkg.NewFunc("modes", sig, ssa.InGo)
	b := fn.MakeBody(1)

	str := fn.Param(0)
	slice := fn.Param(1)
	array := fn.Param(2)
	low := fn.Param(3)
	high := fn.Param(4)
	max := fn.Param(5)
	none := ssa.Expr{}

	b.Index(str, low, nil)
	b.IndexAddr(slice, low)
	b.Slice(str, low, high, none)
	b.Slice(slice, low, high, none)
	b.Slice(slice, low, high, max)
	b.Slice(array, none, none, none)
	b.SliceToArrayPointer(slice, prog.Type(byteArrayPtr, ssa.InGo))
	b.Return()
	b.EndBuild()
	return pkg.String()
}
