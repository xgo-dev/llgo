package ssa

import (
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestAbiStructFieldsUseGo386Offsets(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "386"))

	st := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "I", types.Typ[types.Int], false),
		types.NewField(token.NoPos, nil, "F", types.Typ[types.Float64], false),
	}, nil)
	got := (&aBuilder{Prog: prog}).abiStructFieldOffsets(st)
	want := []int64{0, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Windows 386 reflected field offsets = %v, want %v", got, want)
	}
}

func TestGo386AtomicCmpXchgUsesNativeLLVMResult(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "386"))
	pkg := prog.NewPackage("example.com/p", "p")
	b := pkg.NewFunc("example.com/p.f", NoArgsNoRet, GoDecl).MakeBody(1)
	valueType := prog.Uint64()
	ptr := b.Alloc(valueType, false)
	result := b.AtomicCmpXchg(ptr, prog.IntVal(0, valueType), prog.IntVal(1, valueType))
	_ = b.Extract(result, 1)
	b.Return()
	b.EndBuild()
	if ir := pkg.String(); !strings.Contains(ir, "extractvalue { i64, i1 }") {
		t.Fatalf("Windows/386 cmpxchg result was not extracted from its native LLVM type:\n%s", ir)
	}
}

func TestGo386StructPhysicalLayoutDiffersFromC(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "386"))

	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "I", types.Typ[types.Int], false),
		types.NewField(token.NoPos, nil, "F", types.Typ[types.Float64], false),
	}
	st := types.NewStruct(fields, nil)
	goType := prog.Type(st, GoDecl)
	if got, want := prog.SizeOf(goType), uint64(12); got != want {
		t.Fatalf("Go Windows/386 struct size = %d, want %d", got, want)
	}
	if got, want := prog.OffsetOf(goType, 1), uint64(4); got != want {
		t.Fatalf("Go Windows/386 float64 field offset = %d, want %d", got, want)
	}
	if zero := prog.Zero(goType); zero.impl.Type() != goType.ll {
		t.Fatalf("Go Windows/386 struct zero has type %s, want %s", zero.impl.Type(), goType.ll)
	}
	other := &aType{
		ll:  goType.ll,
		raw: rawType{Type: types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))},
	}
	if _, ok := prog.structLayout(other); ok {
		t.Fatal("LLVM-identical type inherited layout metadata from a different Go type")
	}

	pkg := types.NewPackage("example.com/c", "c")
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Pair", nil), st, nil)
	prog.SetTypeCallConv("example.com/c.Pair", CDecl)
	cType := prog.Type(named, CDecl)
	if got, want := prog.SizeOf(cType), uint64(16); got != want {
		t.Fatalf("C Windows/386 struct size = %d, want %d", got, want)
	}
	if got, want := prog.OffsetOf(cType, 1), uint64(8); got != want {
		t.Fatalf("C Windows/386 float64 field offset = %d, want %d", got, want)
	}
}

func TestGo386StructLayoutEdgeCases(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()

	t.Run("single float64 retains Go alignment", func(t *testing.T) {
		st := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "F", types.Typ[types.Float64], false),
		}, nil)
		typ := prog.Type(st, GoDecl)
		if got, want := prog.SizeOf(typ), uint64(8); got != want {
			t.Fatalf("Go Windows/386 struct size = %d, want %d", got, want)
		}
		if got, want := prog.AlignOf(typ), uint64(4); got != want {
			t.Fatalf("Go Windows/386 struct alignment = %d, want %d", got, want)
		}
		if got, want := typ.ll.StructElementTypes(), 2; len(got) != want {
			t.Fatalf("LLVM struct element count = %d, want %d", len(got), want)
		}
	})

	t.Run("final zero-sized field has a distinct address", func(t *testing.T) {
		empty := types.NewStruct(nil, nil)
		st := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "I", types.Typ[types.Int32], false),
			types.NewField(token.NoPos, nil, "Z", empty, false),
		}, nil)
		typ := prog.Type(st, GoDecl)
		if got, want := prog.SizeOf(typ), uint64(8); got != want {
			t.Fatalf("Go Windows/386 struct size = %d, want %d", got, want)
		}
		if got, want := prog.OffsetOf(typ, 1), uint64(4); got != want {
			t.Fatalf("zero-sized field offset = %d, want %d", got, want)
		}
	})

	t.Run("named zero constant uses its physical layout", func(t *testing.T) {
		st := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "F", types.Typ[types.Float64], false),
		}, nil)
		pkg := types.NewPackage("example.com/p", "p")
		named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Value", nil), st, nil)
		typ := prog.Type(named, GoDecl)
		if zero := prog.Zero(typ); zero.impl.Type() != typ.ll {
			t.Fatalf("named Go Windows/386 struct zero has type %s, want %s", zero.impl.Type(), typ.ll)
		}
	})
}

func TestGo386TupleUsesGoStructLayout(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "386"))

	results := types.NewTuple(
		types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64]),
		types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]),
		types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64]),
	)
	tupleType := prog.rawType(results)
	if got, want := prog.SizeOf(tupleType), uint64(20); got != want {
		t.Fatalf("Go Windows/386 tuple size = %d, want %d", got, want)
	}
	if got, want := prog.OffsetOf(tupleType, 2), uint64(12); got != want {
		t.Fatalf("Go Windows/386 tuple third result offset = %d, want %d", got, want)
	}

	sig := types.NewSignatureType(nil, nil, nil, nil, results, false)
	if got, want := prog.toLLVMFunc(sig).ReturnType(), tupleType.ll; got != want {
		t.Fatalf("function result type = %s, want tuple type %s", got, want)
	}
	equivalentStruct := prog.Struct(
		prog.Int64(),
		prog.Int32(),
		prog.Int64(),
	)
	if got, want := equivalentStruct.ll, tupleType.ll; got != want {
		t.Fatalf("multi-result value type = %s, want function tuple type %s", got, want)
	}
}

func TestConstStructValueAsUsesDestinationLLVMType(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "386"))

	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "B", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, nil, "U", types.Typ[types.Uint64], false),
	}
	st := types.NewStruct(fields, nil)
	pkg := types.NewPackage("example.com/p", "p")
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Value", nil), st, nil)
	typ := prog.Type(named, GoDecl)
	destination := prog.ctx.StructCreateNamed("example.com/p.destination")
	destination.StructSetBody(typ.ll.StructElementTypes(), typ.ll.IsStructPacked())

	value := prog.constStructValueAs(typ, destination, []llvm.Value{
		prog.IntVal(1, prog.Byte()).impl,
		prog.IntVal(2, prog.Uint64()).impl,
	})
	if value.Type() != destination {
		t.Fatalf("constant type = %s, want exact destination type %s", value.Type(), destination)
	}
	for i, want := range destination.StructElementTypes() {
		if got := value.Operand(i).Type(); got != want {
			t.Errorf("field %d constant type = %s, want %s", i, got, want)
		}
	}
}
