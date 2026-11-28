package fmtspecialize

import (
	"go/constant"
	"go/types"
	"golang.org/x/tools/go/ssa"
	"testing"
)

func TestBuiltinKindsThroughChangeType(t *testing.T) {
	boxed := &ssa.MakeInterface{X: ssa.NewConst(constant.MakeString("x"), types.Typ[types.String])}
	converted := &ssa.ChangeType{X: boxed}
	got, ok := builtinValueKinds(converted, nil, make(map[ssa.Value]bool))
	if !ok || got != builtinKindMask(types.String) {
		t.Fatalf("operand mask = %d, known = %v", got, ok)
	}
}
