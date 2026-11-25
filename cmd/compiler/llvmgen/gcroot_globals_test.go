package llvmgen

import (
	"github.com/xgo-dev/llgo/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmGlobalScanPolicyUsesGoTypes(t *testing.T) {
	for _, target := range []Target{
		{GOOS: "js", GOARCH: "wasm"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "linux", GOARCH: "amd64"},
	} {
		for _, roots := range []bool{false, true} {
			prog := NewProgram(&target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			if target.GOARCH == "wasm" {
				prog.TypeSizes(&types.StdSizes{WordSize: 8, MaxAlign: 8})
			}
			prog.EnableGCRoots(roots)
			pkg := prog.NewPackage("globals", "globals")
			pointer := types.NewPointer(types.Typ[types.Int])
			for _, tc := range []struct {
				name   string
				typ    types.Type
				noscan bool
			}{
				{"scalar", types.Typ[types.Uint64], true},
				{"packed", types.NewArray(types.Typ[types.Uint16], 128), true},
				{"numbers", types.NewStruct([]*types.Var{types.NewVar(0, nil, "X", types.Typ[types.Int64])}, nil), true},
				{"pointer", pointer, false},
				{"pointers", types.NewArray(pointer, 2), false},
				{"mixed", types.NewStruct([]*types.Var{types.NewVar(0, nil, "X", types.Typ[types.Int64]), types.NewVar(0, nil, "P", pointer)}, nil), false},
				{"slice", types.NewSlice(types.Typ[types.Byte]), false},
				{"string", types.Typ[types.String], false},
				{"unsafe", types.Typ[types.UnsafePointer], false},
				{"tuple", types.NewTuple(types.NewVar(0, nil, "X", types.Typ[types.Int])), false},
			} {
				name := "globals." + tc.name
				global := pkg.NewVar(name, types.NewPointer(tc.typ), GoDecl)
				global.InitNil()
				if got := global.impl.Section(); got != "" {
					t.Fatal("zero storage moved out of bss", got)
				}
				global.impl.SetInitializer(globalScanTestValue(global.impl.GlobalValueType()))
				global.excludePointerFreeGlobal()
				got := pkg.Module().NamedGlobal(name).Section()
				want := ""
				if roots && target.GOARCH == "wasm" && tc.noscan {
					want = WasmGCNoScanSection
				}
				if got != want {
					t.Fatalf("%s/%s roots=%v %s: section=%q want %q", target.GOOS, target.GOARCH, roots, tc.name, got, want)
				}
			}
		}
	}
}

// Nonzero storage, including widened pointer slots, exercises the Go type
// decision independently of LLVM's final integer representation.
func globalScanTestValue(typ llvm.Type) llvm.Value {
	switch typ.TypeKind() {
	case llvm.IntegerTypeKind:
		return llvm.ConstInt(typ, 1, false)
	case llvm.PointerTypeKind:
		return llvm.ConstIntToPtr(llvm.ConstInt(typ.Context().Int64Type(), 1, false), typ)
	case llvm.ArrayTypeKind:
		values := make([]llvm.Value, typ.ArrayLength())
		for i := range values {
			values[i] = globalScanTestValue(typ.ElementType())
		}
		return llvm.ConstArray(typ.ElementType(), values)
	case llvm.StructTypeKind:
		fields := typ.StructElementTypes()
		values := make([]llvm.Value, len(fields))
		for i := range fields {
			values[i] = globalScanTestValue(fields[i])
		}
		return llvm.ConstNamedStruct(typ, values)
	}
	panic("unexpected global storage type")
}

func TestWasmStaticByteStorageKeepsZeroDataInBSS(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "js", GOARCH: "wasm"})
	defer prog.Dispose()
	prog.EnableGCRoots(true)
	pkg := prog.NewPackage("bytes", "bytes")
	for _, tc := range []struct {
		data    []byte
		section string
	}{
		{[]byte{1, 2, 3}, WasmGCNoScanSection},
		{make([]byte, 1024), ""},
	} {
		ptr := pkg.createGlobalBytes(tc.data)
		if ptr.IsAGlobalValue().IsNil() {
			ptr = ptr.Operand(0)
		}
		if got := ptr.Section(); got != tc.section {
			t.Fatalf("static byte storage section = %q, want %q", got, tc.section)
		}
	}
}
