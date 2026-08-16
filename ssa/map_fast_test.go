package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestMapKeyFastKind(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	ptrSize := prog.PointerSize()

	namedUint32 := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "ID", nil),
		types.Typ[types.Uint32],
		nil,
	)
	namedString := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "Name", nil),
		types.Typ[types.String],
		nil,
	)
	structKey := types.NewStruct(
		[]*types.Var{types.NewField(token.NoPos, nil, "value", types.Typ[types.Uint64], false)},
		nil,
	)
	ptrKey := types.NewPointer(types.Typ[types.Int])
	chanKey := types.NewChan(types.SendRecv, types.Typ[types.Int])

	tests := []struct {
		name string
		key  types.Type
		want mapFastKind
	}{
		{"uint32", types.Typ[types.Uint32], mapFast32},
		{"named uint32", namedUint32, mapFast32},
		{"int32", types.Typ[types.Int32], mapFast32},
		{"uint64", types.Typ[types.Uint64], mapFast64},
		{"int64", types.Typ[types.Int64], mapFast64},
		{"int", types.Typ[types.Int], mapFast64},
		{"uintptr", types.Typ[types.Uintptr], mapFast64},
		{"string", types.Typ[types.String], mapFastStr},
		{"named string", namedString, mapFastStr},
		{"unsafe pointer", types.Typ[types.UnsafePointer], mapFast64Ptr},
		{"pointer", ptrKey, mapFast64Ptr},
		{"channel", chanKey, mapFast64Ptr},
		{"float32 fallback", types.Typ[types.Float32], mapFastNone},
		{"float64 fallback", types.Typ[types.Float64], mapFastNone},
		{"struct fallback", structKey, mapFastNone},
	}
	if ptrSize == 4 {
		for i := range tests {
			switch tests[i].want {
			case mapFast64:
				if tests[i].key == types.Typ[types.Int] || tests[i].key == types.Typ[types.Uintptr] {
					tests[i].want = mapFast32
				}
			case mapFast64Ptr:
				tests[i].want = mapFast32Ptr
			}
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapType := types.NewMap(test.key, types.Typ[types.Int])
			if got := mapKeyFastKind(prog, mapType); got != test.want {
				t.Fatalf("mapKeyFastKind(map[%v]int) = %v, want %v", test.key, got, test.want)
			}
		})
	}
}

func TestMapKeyFastKindLargeElemFallback(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()

	largeElem := types.NewArray(types.Typ[types.Uint64], 17)
	for _, key := range []types.Type{
		types.Typ[types.Uint32],
		types.Typ[types.Uint64],
		types.Typ[types.String],
		types.NewPointer(types.Typ[types.Int]),
	} {
		mapType := types.NewMap(key, largeElem)
		if got := mapKeyFastKind(prog, mapType); got != mapFastNone {
			t.Errorf("mapKeyFastKind(map[%v][17]uint64) = %v, want mapFastNone", key, got)
		}
	}
}

func TestMapKeyFastKind32BitTarget(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	if got := prog.PointerSize(); got != 4 {
		t.Fatalf("PointerSize() = %d, want 4", got)
	}

	for _, test := range []struct {
		name string
		key  types.Type
		want mapFastKind
	}{
		{"int", types.Typ[types.Int], mapFast32},
		{"uintptr", types.Typ[types.Uintptr], mapFast32},
		{"unsafe pointer", types.Typ[types.UnsafePointer], mapFast32Ptr},
		{"pointer", types.NewPointer(types.Typ[types.Int]), mapFast32Ptr},
		{"channel", types.NewChan(types.SendRecv, types.Typ[types.Int]), mapFast32Ptr},
	} {
		t.Run(test.name, func(t *testing.T) {
			mapType := types.NewMap(test.key, types.Typ[types.Int])
			if got := mapKeyFastKind(prog, mapType); got != test.want {
				t.Fatalf("mapKeyFastKind(map[%v]int) = %v, want %v", test.key, got, test.want)
			}
		})
	}

	pkg := prog.NewPackage("p", "example.com/p")
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "key", types.NewPointer(types.Typ[types.Int])))
	fn := pkg.NewFunc("access", types.NewSignatureType(nil, nil, nil, params, nil, false), InGo)
	b := fn.MakeBody(1)
	arg := b.mapKeyAccessArg(Expr{}, fn.Param(0), mapFast32Ptr)
	if arg.Type != prog.Uint32() {
		t.Fatalf("32-bit pointer access argument type = %v, want uint32", arg.Type)
	}
	b.Return()
	if ir := pkg.String(); !strings.Contains(ir, "ptrtoint ptr %0 to i32") {
		t.Fatalf("32-bit pointer access did not emit ptrtoint:\n%s", ir)
	}
}

func TestMapKeyFastKindNonMapFallback(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	if got := mapKeyFastKind(prog, types.Typ[types.Int]); got != mapFastNone {
		t.Fatalf("mapKeyFastKind(int) = %v, want mapFastNone", got)
	}
}

func TestMapFastRuntimeNames(t *testing.T) {
	tests := []struct {
		kind       mapFastKind
		access1    string
		access2    string
		assign     string
		deleteName string
	}{
		{mapFastNone, "MapAccess1", "MapAccess2", "MapAssign", "MapDelete"},
		{mapFast32, "MapAccess1Fast32", "MapAccess2Fast32", "MapAssignFast32", "MapDeleteFast32"},
		{mapFast64, "MapAccess1Fast64", "MapAccess2Fast64", "MapAssignFast64", "MapDeleteFast64"},
		{mapFast32Ptr, "MapAccess1Fast32", "MapAccess2Fast32", "MapAssignFast32Ptr", "MapDeleteFast32"},
		{mapFast64Ptr, "MapAccess1Fast64", "MapAccess2Fast64", "MapAssignFast64Ptr", "MapDeleteFast64"},
		{mapFastStr, "MapAccess1FastStr", "MapAccess2FastStr", "MapAssignFastStr", "MapDeleteFastStr"},
	}

	for _, test := range tests {
		if got := test.kind.accessName(false); got != test.access1 {
			t.Errorf("%v access1 = %q, want %q", test.kind, got, test.access1)
		}
		if got := test.kind.accessName(true); got != test.access2 {
			t.Errorf("%v access2 = %q, want %q", test.kind, got, test.access2)
		}
		if got := test.kind.assignName(); got != test.assign {
			t.Errorf("%v assign = %q, want %q", test.kind, got, test.assign)
		}
		if got := test.kind.deleteName(); got != test.deleteName {
			t.Errorf("%v delete = %q, want %q", test.kind, got, test.deleteName)
		}
	}
}
