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
				slice := b.MakeSlice(prog.Type(types.NewSlice(tc.typ), GoDecl), prog.Val(1), prog.Val(2))
				b.BuiltinCall("append", slice, slice)
				b.Return()
				b.EndBuild()
				ir := fn.impl.String()
				zero, uninit, flag := "AllocNoScanU", "AllocNoScanU", "i1 false"
				if tc.pointers {
					zero, uninit, flag = "AllocZ", "AllocU", "i1 true"
				}
				if !tc.pointers && !strings.Contains(ir, "@llvm.memset") {
					t.Fatalf("%s missing zero initialization:\n%s", tc.name, ir)
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
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
