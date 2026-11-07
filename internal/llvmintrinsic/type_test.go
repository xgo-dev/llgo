package llvmintrinsic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestTypePatterns(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i8, i32, f32 := ctx.Int8Type(), ctx.Int32Type(), ctx.FloatType()
	v4 := llvm.VectorType(i32, 4)
	ptr := llvm.PointerType(i8, 0)
	arr := llvm.ArrayType(i8, 3)
	st := ctx.StructType([]llvm.Type{arr, i32}, false)
	ctx2 := llvm.NewContext()
	defer ctx2.Dispose()
	for _, tc := range []struct {
		name    string
		pattern Type
		actual  llvm.Type
		ok      bool
	}{
		{"integer", I8, i8, true}, {"integer-width", I8, i32, false},
		{"integer-kind", I32, f32, false}, {"integer-scalar", Integer, v4, false},
		{"integer-vector", Integers, v4, true}, {"float-vector", Floats, llvm.VectorType(f32, 4), true},
		{"float-scalar", Floating, llvm.VectorType(f32, 4), false}, {"float-kind", F64, f32, false},
		{"vector", AnyVector, v4, true}, {"not-vector", AnyVector, arr, false},
		{"opaque-pointer", AnyPointer, ptr, true}, {"pointer-space", Ptr(1), ptr, false},
		{"pointer-not-integer", AnyPointer, ctx.Int64Type(), false},
		{"array", Array(3, I8), arr, true}, {"array-length", Array(2, I8), arr, false},
		{"array-element", Array(3, I32), arr, false}, {"array-not-vector", Array(3, I8), llvm.VectorType(i8, 3), false},
		{"nested-array", Array(2, Array(3, I8)), llvm.ArrayType(arr, 2), true},
		{"zero-array", Array(0, I8), llvm.ArrayType(i8, 0), true},
		{"struct", Struct(Array(3, I8), I32), st, true},
		{"packing", PackedStruct(Array(3, I8), I32), st, false},
		{"struct-arity", Struct(I32), st, false},
		{"struct-field", Struct(Array(3, I32), I32), st, false},
		{"opaque-struct", Struct(), ctx.StructCreateNamed("opaque"), false},
		{"exact", Exact(st), st, true}, {"exact-other-context", Exact(i32), ctx2.Int32Type(), false},
		{"vector-fixed", Vector(Fixed(4), I32), v4, true},
		{"vector-scalable", Vector(Scalable(4), I32), vectorType(i32, 4, true), true},
		{"scalable-distinct", Vector(Fixed(4), I32), vectorType(i32, 4, true), false},
		{"vector-lanes", Vector(Fixed(2), I32), v4, false},
		{"vector-element", Vector(Fixed(4), I8), v4, false},
		{"vector-zero", Vector(Fixed(0), I32), v4, false},
		{"vector-overflow", Vector(Fixed(1<<32), I32), v4, false},
		{"vector-nil-count", Vector(nil, I32), v4, false},
		{"nil-actual", Any, llvm.Type{}, false},
		{"nil-exact", Exact(llvm.Type{}), i32, false},
		{"unbound", Ref("unknown"), i32, false},
		{"bad-binding", Bind("$arg0", Any), i32, false},
		{"empty-binding", Bind("", Any), i32, false},
		{"nil-predicate", Where(Any, "nil", nil), i32, false},
		{"predicate", Where(Integer, "even bytes", func(_ Context, t llvm.Type) error {
			bits, _ := Bits(t)
			if bits%16 != 0 {
				return fmt.Errorf("bit width %d", bits)
			}
			return nil
		}), i32, true},
		{"predicate-fail", Where(Integer, "even bytes", func(_ Context, t llvm.Type) error {
			bits, _ := Bits(t)
			if bits%16 != 0 {
				return fmt.Errorf("bit width %d", bits)
			}
			return nil
		}), i8, false},
		{"predicate-base", Where(Integer, "integer", func(Context, llvm.Type) error { return nil }), ptr, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Match(tc.pattern, tc.actual)
			if (err == nil) != tc.ok {
				t.Fatalf("pattern %v, type %v: %v", tc.pattern, tc.actual, err)
			}
		})
	}
}

func TestForwardTypeReferences(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i32 := ctx.Int32Type()
	v4 := llvm.VectorType(i32, 4)
	pattern := Func(Elem(Ref("T")), Bind("T", AnyVector))
	if _, err := Match(pattern, llvm.FunctionType(i32, []llvm.Type{v4}, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := Match(pattern, llvm.FunctionType(ctx.Int16Type(), []llvm.Type{v4}, false)); err == nil {
		t.Fatal("accepted wrong element")
	}
	for _, pattern := range []Type{
		Func(Ref("T"), Bind("T", Integer), Bind("T", Integer)),
		Func(Where(Any, "deferred", func(c Context, t llvm.Type) error {
			want, err := c.Resolve(Ref("T"))
			if err != nil {
				return err
			}
			if want != t {
				return fmt.Errorf("wrong type")
			}
			return nil
		}), Bind("T", Integer), Ref("T")),
	} {
		if _, err := Match(pattern, llvm.FunctionType(i32, []llvm.Type{i32, i32}, false)); err != nil {
			t.Fatal(err)
		}
		if _, err := Match(pattern, llvm.FunctionType(i32, []llvm.Type{i32, ctx.Int16Type()}, false)); err == nil {
			t.Fatal("inconsistent reference accepted")
		}
	}
	for _, pattern := range []Type{Elem(Exact(i32)), Elem(Exact(llvm.PointerType(i32, 0))), Shape(Ref("missing"), I1), Vector(Lanes(Exact(i32)), I32), Vector(Divide(nil, 2), I32), Vector(Divide(Fixed(4), 0), I32), Vector(Divide(Fixed(2), 4), I32)} {
		if _, err := Match(pattern, v4); err == nil {
			t.Fatalf("accepted %v", pattern)
		}
	}
	if _, err := Match(Elem(Exact(llvm.ArrayType(i32, 2))), i32); err != nil {
		t.Fatal(err)
	}
	if _, err := Match(Func(Shape(Arg(0), I1), Integer), llvm.FunctionType(ctx.Int1Type(), []llvm.Type{i32}, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := Match(Func(Shape(Arg(0), I1), Integer), llvm.FunctionType(llvm.VectorType(ctx.Int1Type(), 1), []llvm.Type{i32}, false)); err == nil {
		t.Fatal("scalar/vector shape conflated")
	}
	if _, err := Match(Func(Shape(Arg(0), I1), AnyVector), llvm.FunctionType(llvm.VectorType(ctx.Int1Type(), 2), []llvm.Type{v4}, false)); err == nil {
		t.Fatal("vector lane mismatch")
	}
}

func TestBitWidthsAndPointerIndexLayout(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	for _, typ := range []llvm.Type{ctx.Int32Type(), ctx.FloatType(), llvm.VectorType(ctx.FloatType(), 4)} {
		if bits, err := Bits(typ); err != nil || bits != 32 {
			t.Fatalf("bits(%v)=%d,%v", typ, bits, err)
		}
	}
	for _, typ := range []llvm.Type{llvm.Type{}, llvm.PointerType(ctx.Int8Type(), 0), ctx.StructType(nil, false)} {
		if _, err := Bits(typ); err == nil {
			t.Fatalf("accepted %v", typ)
		}
	}
	ptr := llvm.PointerType(ctx.Int8Type(), 1)
	layout := "e-p:64:64-p1:64:64:64:32"
	for _, typ := range []llvm.Type{ptr, llvm.VectorType(ptr, 4)} {
		if bits, err := PointerIndexBits(typ, layout); err != nil || bits != 32 {
			t.Fatalf("index bits=%d,%v", bits, err)
		}
	}
	for _, tc := range []struct {
		typ    llvm.Type
		layout string
	}{{llvm.Type{}, layout}, {ctx.Int32Type(), layout}, {ptr, "not-a-layout"}} {
		if _, err := PointerIndexBits(tc.typ, tc.layout); err == nil {
			t.Fatal("accepted bad pointer/layout")
		}
	}
	mask := Where(Integer, "pointer index width", func(c Context, t llvm.Type) error {
		p, err := c.Resolve(Arg(0))
		if err != nil {
			return err
		}
		width, err := PointerIndexBits(p, c.DataLayout())
		if err != nil {
			return err
		}
		bits, _ := Bits(t)
		if bits != width {
			return fmt.Errorf("expected %d bits", width)
		}
		return nil
	})
	if _, err := Match(Func(Ref("P"), Bind("P", AnyPointer), mask), llvm.FunctionType(ptr, []llvm.Type{ptr, ctx.Int32Type()}, false), WithDataLayout(layout)); err != nil {
		t.Fatal(err)
	}
	_, err := Match(Func(Ref("P"), Bind("P", AnyPointer), mask), llvm.FunctionType(ptr, []llvm.Type{ptr, ctx.Int64Type()}, false), WithDataLayout(layout))
	if err == nil || !strings.Contains(err.Error(), "32 bits") {
		t.Fatalf("got %v", err)
	}
}
