package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestTypedAllocatorSelection(t *testing.T) {
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
			pkg := prog.NewPackage("p", "example.com/typedalloc")
			vectorPkg := types.NewPackage("simd/archsimd", "archsimd")
			align16 := types.NewNamed(types.NewTypeName(0, vectorPkg, "v128", nil), types.NewStruct([]*types.Var{
				types.NewField(0, vectorPkg, "tag", types.NewArray(types.Typ[types.Byte], 0), false),
			}, nil), nil)
			alignedRecord := func(value types.Type) types.Type {
				return types.NewStruct([]*types.Var{
					types.NewField(0, nil, "Tag", align16, false),
					types.NewField(0, nil, "Value", value, false),
				}, nil)
			}
			for _, tc := range []struct {
				name     string
				typ      types.Type
				pointers bool
			}{
				{"byte", types.Typ[types.Byte], false},
				{"rune", types.Typ[types.Rune], false},
				{"float", types.Typ[types.Float64], false},
				{"uintptr", types.Typ[types.Uintptr], false},
				{"string", types.Typ[types.String], true},
				{"pointer", types.NewPointer(types.Typ[types.Int]), true},
				{"unsafePointer", types.Typ[types.UnsafePointer], true},
				{"sliceHeader", types.NewSlice(types.Typ[types.Byte]), true},
				{"closure", NoArgsNoRet, true},
				{"interface", types.NewInterfaceType(nil, nil).Complete(), true},
				{"struct", types.NewStruct([]*types.Var{types.NewField(0, nil, "V", types.NewArray(types.Typ[types.Float64], 3), false)}, nil), false},
				{"pointerStruct", types.NewStruct([]*types.Var{types.NewField(0, nil, "V", types.NewArray(types.Typ[types.String], 3), false)}, nil), true},
				{"alignedScalar", alignedRecord(types.NewArray(types.Typ[types.Uint64], 2)), false},
				{"alignedPointer", alignedRecord(types.NewPointer(types.Typ[types.Int])), true},
				{"emptyPointerArray", types.NewStruct([]*types.Var{
					types.NewField(0, nil, "Empty", types.NewArray(types.NewPointer(types.Typ[types.Int]), 0), false),
					types.NewField(0, nil, "N", types.Typ[types.Int], false),
				}, nil), false},
			} {
				fn := pkg.NewFunc("p."+tc.name, NoArgsNoRet, GoDecl)
				b := fn.MakeBody(1)
				elem := prog.Type(tc.typ, GoDecl)
				b.Alloc(elem, true)
				b.AllocU(elem)
				sliceType := prog.Type(types.NewSlice(tc.typ), GoDecl)
				slice := b.MakeSlice(sliceType, prog.Val(1), prog.Val(2))
				b.MakeSliceEx(sliceType, prog.Val(1), prog.Val(2), false)
				if target.GOARCH == "386" {
					b.MakeSliceEx(sliceType, prog.IntVal(1, prog.Int64()), prog.IntVal(2, prog.Int64()), false)
				}
				b.BuiltinCall("append", slice, slice)
				b.Return()
				b.EndBuild()
				ir := fn.impl.String()
				zero, uninit, flag := "AllocAtomicU", "AllocAtomicU", "i1 false"
				if tc.pointers {
					zero, uninit, flag = "AllocZ", "AllocU", "i1 true"
				}
				if strings.HasPrefix(tc.name, "aligned") {
					if got := prog.AlignOf(elem); got != 16 {
						t.Fatalf("%s alignment = %d, want 16", tc.name, got)
					}
					zero += "Aligned"
					uninit += "Aligned"
				}
				if got := strings.Contains(ir, "@llvm.memset"); got != !tc.pointers {
					t.Fatalf("%s inline zero initialization = %v, want %v:\n%s", tc.name, got, !tc.pointers, ir)
				}
				for _, name := range []string{zero, uninit} {
					if !strings.Contains(ir, PkgRuntime+"."+name+"\"") {
						t.Fatalf("%s missing %s:\n%s", tc.name, name, ir)
					}
				}
				for _, line := range strings.Split(ir, "\n") {
					if strings.Contains(line, "call") && (strings.Contains(line, ".MakeSlice") || strings.Contains(line, ".SliceAppend")) && !strings.Contains(line, flag+")") {
						t.Fatalf("%s slice allocation has wrong pointer flag: %s", tc.name, line)
					}
				}
			}
			fn := pkg.NewFunc("p.appendString", NoArgsNoRet, GoDecl)
			b := fn.MakeBody(1)
			bytes := b.MakeSlice(prog.Type(types.NewSlice(types.Typ[types.Byte]), GoDecl), prog.Val(0), prog.Val(0))
			b.BuiltinCall("append", bytes, b.Str("text"))
			b.Return()
			b.EndBuild()
			ir := fn.impl.String()
			if !strings.Contains(ir, PkgRuntime+".SliceAppend\"") {
				t.Fatalf("string append missing SliceAppend:\n%s", ir)
			}
			for _, line := range strings.Split(ir, "\n") {
				if strings.Contains(line, ".SliceAppend\"") && !strings.Contains(line, "i1 false)") {
					t.Fatalf("string append has wrong pointer flag: %s", line)
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
