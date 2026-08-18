//go:build go1.25

package types_test

import (
	"go/token"
	"go/types"
	"testing"
)

func TestVarKind(t *testing.T) {
	variable := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])
	if got := variable.Kind(); got != types.PackageVar {
		t.Fatalf("new variable kind = %v, want %v", got, types.PackageVar)
	}
	variable.SetKind(types.LocalVar)
	if got := variable.Kind(); got != types.LocalVar {
		t.Fatalf("updated variable kind = %v, want %v", got, types.LocalVar)
	}
	if got := types.LocalVar.String(); got != "LocalVar" {
		t.Fatalf("LocalVar.String = %q", got)
	}
}

func TestLookupSelection(t *testing.T) {
	field := types.NewField(token.NoPos, nil, "Value", types.Typ[types.String], false)
	typ := types.NewStruct([]*types.Var{field}, nil)
	selection, ok := types.LookupSelection(typ, true, nil, "Value")
	if !ok {
		t.Fatal("LookupSelection did not find Value")
	}
	if selection.Obj() != field || selection.Indirect() {
		t.Fatalf("LookupSelection = (%v, indirect %v)", selection.Obj(), selection.Indirect())
	}
	if index := selection.Index(); len(index) != 1 || index[0] != 0 {
		t.Fatalf("Selection.Index = %v", index)
	}
	if _, ok := types.LookupSelection(typ, true, nil, "Missing"); ok {
		t.Fatal("LookupSelection unexpectedly found Missing")
	}
}
