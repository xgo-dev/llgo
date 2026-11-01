//go:build !llgo

package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"math"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Ordinary Go types must not add allocations beyond the original Go sizes hooks.
func TestOrdinarySourceLayoutAllocations(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	base := types.SizesFor("gc", "amd64")
	sizes := prog.TypeSizes(base)
	field := types.NewField(token.NoPos, nil, "x", types.Typ[types.Int64], false)
	nested := types.NewStruct([]*types.Var{field}, nil)
	fields := make([]*types.Var, 128)
	for i := range fields {
		fields[i] = types.NewField(token.NoPos, nil, fmt.Sprintf("f%d", i), nested, false)
	}
	ordinary := types.NewStruct(fields, nil)
	for _, tc := range []struct {
		name string
		call func(types.Sizes)
	}{
		{"size", func(s types.Sizes) {
			if s.Sizeof(ordinary) != 1024 {
				t.Fatal("size")
			}
		}},
		{"alignment", func(s types.Sizes) {
			if s.Alignof(ordinary) != 8 {
				t.Fatal("alignment")
			}
		}},
		{"offsets", func(s types.Sizes) {
			if s.Offsetsof(fields)[127] != 1016 {
				t.Fatal("offset")
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := testing.AllocsPerRun(10, func() { tc.call(base) })
			if got := testing.AllocsPerRun(10, func() { tc.call(sizes) }); got > want {
				t.Fatalf("allocations = %g, baseline = %g", got, want)
			}
		})
	}
}

func TestOrdinaryNamedLayoutAllocations(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	base := types.SizesFor("gc", "amd64")
	sizes := prog.TypeSizes(base)
	pkg := types.NewPackage("example.com/ordinary/package/path", "ordinary")
	named := func(name string, underlying types.Type) *types.Named {
		return types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), underlying, nil)
	}
	integer := named("Integer", types.Typ[types.Int64])
	record := named("Record", types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", integer, false),
	}, nil))
	nested := named("Nested", types.NewArray(record, 8))
	for _, metadata := range []bool{false, true} {
		if metadata {
			// An unrelated native declaration must not make ordinary named
			// layout queries construct linknames and consult the metadata map.
			prog.SetTypeCallConv("other.Native", CDecl)
		}
		for _, typ := range []types.Type{integer, record, nested} {
			t.Run(fmt.Sprintf("%s/native-metadata=%t", typ, metadata), func(t *testing.T) {
				want := testing.AllocsPerRun(100, func() { base.Alignof(typ) })
				got := testing.AllocsPerRun(100, func() { sizes.Alignof(typ) })
				if got > want {
					t.Fatalf("Alignof allocations = %g, baseline = %g", got, want)
				}
				if sizes.Alignof(typ) != base.Alignof(typ) || sizes.Sizeof(typ) != base.Sizeof(typ) {
					t.Fatal("ordinary named layout changed")
				}
			})
		}
	}
}

func TestFreshProgramVectorLayout(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64", "386"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: arch})
			defer prog.Dispose()
			// Standalone SSA API users need not call TypeSizes first.
			_, vector := layoutTestVector()
			typ := prog.Type(vector, GoDecl)
			if prog.SizeOf(typ) != 16 || prog.AlignOf(typ) != 16 {
				t.Fatal("fresh vector layout")
			}
			box := types.NewStruct([]*types.Var{
				types.NewField(token.NoPos, nil, "Tag", types.Typ[types.Byte], false),
				types.NewField(token.NoPos, nil, "Value", vector, false),
			}, nil)
			boxType := prog.Type(box, GoDecl)
			if prog.SizeOf(boxType) != 32 || prog.AlignOf(boxType) != 16 || prog.OffsetOf(boxType, 1) != 16 {
				t.Fatal("fresh containing struct layout")
			}
		})
	}
}

func TestNativeVectorFieldOffsets(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64", "386"} {
		for _, sourcePositions := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/positions=%t", arch, sourcePositions), func(t *testing.T) {
				prog := NewProgram(&Target{GOOS: "linux", GOARCH: arch})
				defer prog.Dispose()
				sizes := prog.TypeSizes(types.SizesFor("gc", arch))
				_, vector := layoutTestVector()
				pkg := types.NewPackage("example.com/native", "native")
				fields := []*types.Var{
					types.NewField(10, pkg, "Vector", vector, false),
					types.NewField(11, pkg, "Callback", NoArgsNoRet, false),
					types.NewField(12, pkg, "Tail", types.Typ[types.Byte], false),
				}
				if sourcePositions {
					prog.SetTypeCallConv("example.com/native.Native", CDecl)
					prog.SetNativeStructFields([]token.Pos{10, 11, 12})
				} else {
					for i, f := range fields {
						fields[i] = types.NewField(token.NoPos, pkg, f.Name(), f.Type(), false)
					}
					prog.SetTypeCallConv("example.com/native.Native", CDecl, fields...)
				}
				native := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Native", nil), types.NewStruct(fields, nil), nil)
				// Query offsets before any LLVM type or other source layout query.
				offsets := sizes.Offsetsof(fields)
				if offsets[1] != 16 || offsets[2] != int64(16+prog.ptrSize) || sizes.Sizeof(native) != 32 {
					t.Fatalf("native offsets=%v size=%d", offsets, sizes.Sizeof(native))
				}
				storage := prog.rawType(native)
				for i, want := range offsets {
					if got := prog.OffsetOf(storage, i); got != uint64(want) {
						t.Fatalf("field %d: storage=%d source=%d", i, got, want)
					}
				}
				ordinary := []*types.Var{
					types.NewField(token.NoPos, pkg, "Vector", vector, false),
					types.NewField(token.NoPos, pkg, "Callback", NoArgsNoRet, false),
					types.NewField(token.NoPos, pkg, "Tail", types.Typ[types.Byte], false),
				}
				if sizes.Offsetsof(ordinary)[2] != int64(16+2*prog.ptrSize) {
					t.Fatal("unregistered fields lost Go closure layout")
				}
			})
		}
	}
}

func layoutTestVector() (tag, vector *types.Named) {
	pkg := types.NewPackage("simd/archsimd", "archsimd")
	tag = types.NewNamed(types.NewTypeName(token.NoPos, pkg, "v128", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "tag", types.NewArray(types.Typ[types.Byte], 0), false),
	}, nil), nil)
	vector = types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Float32x4", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "tag", tag, false),
		types.NewField(token.NoPos, pkg, "lanes", types.NewArray(types.Typ[types.Float32], 4), false),
	}, nil), nil)
	return
}

func TestVectorSourceLayoutArrayOverflow(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	tag, vector := layoutTestVector()
	sizes := prog.TypeSizes(types.SizesFor("gc", "amd64"))
	for _, tc := range []struct {
		name string
		typ  types.Type
		size int64
	}{
		{"array", types.NewArray(vector, 3), 48},
		{"zero array", types.NewArray(vector, 0), 0},
		{"zero sized elements", types.NewArray(tag, math.MaxInt64), 0},
		{"array overflow", types.NewArray(vector, math.MaxInt64/16+1), -1},
		{"nested overflow", types.NewArray(types.NewArray(vector, math.MaxInt64/16+1), 2), -1},
		{"field overflow", types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "v", vector, false),
			types.NewField(token.NoPos, nil, "large", types.NewArray(types.Typ[types.Byte], math.MaxInt64), false),
		}, nil), -1},
		{"trailing zero field overflow", types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "v", tag, false),
			types.NewField(token.NoPos, nil, "large", types.NewArray(types.Typ[types.Byte], math.MaxInt64), false),
			types.NewField(token.NoPos, nil, "tail", types.NewArray(types.Typ[types.Byte], 0), false),
		}, nil), -1},
		{"alignment overflow", types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "large", types.NewArray(types.Typ[types.Byte], math.MaxInt64), false),
			types.NewField(token.NoPos, nil, "v", tag, false),
		}, nil), -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sizes.Sizeof(tc.typ); got != tc.size {
				t.Fatalf("size = %d, want %d", got, tc.size)
			}
			if got := sizes.Alignof(tc.typ); got != 16 {
				t.Fatalf("alignment = %d, want 16", got)
			}
		})
	}
}

func TestAlignedStorageHelpers(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
	} {
		t.Run(target.GOARCH, func(t *testing.T) {
			t.Setenv("GOOS", target.GOOS)
			t.Setenv("GOARCH", target.GOARCH)
			prog := NewProgram(target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			tag, vector := layoutTestVector()
			typ, tagType := prog.rawType(vector), prog.rawType(tag)
			pkg := prog.NewPackage("p", "p")
			fn := pkg.NewFunc("p.allocate", NoArgsNoRet, GoDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			b.Alloc(typ, true)
			b.AllocU(typ)
			sliceType := prog.rawType(types.NewSlice(vector))
			slice := b.MakeSlice(sliceType, prog.Val(1), prog.Val(2))
			b.BuiltinCall("append", slice, slice)
			b.MakeChan(prog.rawType(types.NewChan(types.SendRecv, vector)), prog.Val(1))
			// Also protect the 64-bit length entry points on a 32-bit target.
			b.MakeSlice(sliceType, prog.IntVal(1, prog.Int64()), prog.IntVal(2, prog.Int64()))
			b.MakeChan(prog.rawType(types.NewChan(types.SendRecv, vector)), prog.IntVal(1, prog.Int64()))
			marker := b.Aggregate(tagType, prog.Zero(prog.Field(tagType, 0)))
			if got := b.Field(marker, 0); !got.impl.IsNull() {
				t.Fatal("zero-sized SIMD marker field is not zero")
			}
			b.Return()
			b.EndBuild()
			ir := fn.impl.String()
			for _, helper := range []string{"AllocZAligned", "AllocUAligned", "MakeSliceAligned", "NewChanAligned", "SliceAppendAligned"} {
				if !strings.Contains(ir, PkgRuntime+"."+helper) {
					t.Fatalf("missing %s call:\n%s", helper, ir)
				}
			}
			if target.GOARCH == "386" {
				for _, helper := range []string{"MakeSlice64Aligned", "NewChan64Aligned"} {
					if !strings.Contains(ir, PkgRuntime+"."+helper) {
						t.Fatalf("missing %s call:\n%s", helper, ir)
					}
				}
			}
			zeroType := prog.rawType(types.NewArray(vector, 0))
			zero := pkg.moduleZeroSizedAlloc(zeroType)
			if zero.impl.Alignment() != 16 || zero.impl.Name() != moduleZeroName+"align16" {
				t.Fatalf("aligned zero-size sentinel: %s", zero.impl.String())
			}
			if pkg.moduleZeroSizedAlloc(zeroType).impl != zero.impl || pkg.moduleZeroSizedAlloc(prog.rawType(types.NewArray(types.Typ[types.Byte], 0))).impl == zero.impl {
				t.Fatal("zero-size sentinel is not reused separately from ordinary storage")
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestVectorMapLoweringRejectsStorage(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	_, vector := layoutTestVector()
	defer func() {
		if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), "map") {
			t.Fatalf("vector map lowering panic = %v", got)
		}
	}()
	prog.rawType(types.NewMap(types.Typ[types.Int], vector))
}

func TestSIMDTagRejectsMalformedMarker(t *testing.T) {
	pkg := types.NewPackage("simd/archsimd", "archsimd")
	for _, raw := range []types.Type{types.Typ[types.Int], types.NewStruct(nil, nil)} {
		tag := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "v128", nil), raw, nil)
		if _, ok := simdTagIndex(tag); ok {
			t.Fatalf("malformed marker accepted: %s", raw)
		}
	}
}

func TestVectorNativeFunctionFieldLayout(t *testing.T) {
	for _, arch := range []string{"amd64", "386"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: arch})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", arch))
			_, vector := layoutTestVector()
			fields := []*types.Var{
				types.NewField(token.NoPos, nil, "vector", vector, false),
				types.NewField(token.NoPos, nil, "callback", NoArgsNoRet, false),
				types.NewField(token.NoPos, nil, "byte", types.Typ[types.Byte], false),
			}
			layout, offsets, changed := (*goProgram)(prog).vectorStructLayout(fields, true)
			wantByte := int64(16 + prog.ptrSize)
			if !changed || layout.size != 32 || layout.align != 16 || len(offsets) != 3 || offsets[0] != 0 || offsets[1] != 16 || offsets[2] != wantByte {
				t.Fatalf("native callback layout = %+v, offsets = %v, changed = %v", layout, offsets, changed)
			}
		})
	}
}
