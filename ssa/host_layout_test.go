package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strconv"
	"testing"
)

func TestHostLayout(t *testing.T) {
	Initialize(InitAll)
	for _, tc := range []struct {
		os, arch, profile string
		size, alignment   int64
		offsets           []int64
	}{
		{"linux", "amd64", "", 40, 8, []int64{0, 0, 8, 16, 24, 32}},
		{"darwin", "arm64", "", 40, 8, []int64{0, 0, 8, 16, 24, 32}},
		{"linux", "386", "", 24, 4, []int64{0, 0, 4, 12, 16, 20}},
		{"windows", "386", "", 32, 8, []int64{0, 0, 8, 16, 20, 24}},
		{"linux", "arm", "", 32, 8, []int64{0, 0, 8, 16, 20, 24}},
		{"wasip1", "wasm", "w32", 40, 8, []int64{0, 0, 8, 16, 24, 32}},
	} {
		t.Run(tc.os+"/"+tc.arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: tc.os, GOARCH: tc.arch, WasmProfile: tc.profile})
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", tc.arch))
			setTestRuntime(t, prog)
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "host.go", `package host
import "structs"
import "unsafe"
type Marker structs.HostLayout
type Alias = structs.HostLayout
type Record struct { _ Marker; B byte; F float64; P *byte; U uintptr; Tail byte }
type Derived Record
type Anonymous = struct { _ Alias; B byte; F float64; P *byte; U uintptr; Tail byte }
type Generic[T any] struct { _ structs.HostLayout; B byte; F float64; P *T; U uintptr; Tail byte }
type Instance = Generic[byte]
type GoTail struct { Value Record; Tail struct{} }
type Outer struct { B byte; Value Record; Tail byte }
type Inner struct { B byte; F float64; P *byte }
type Nested struct { _ structs.HostLayout; Inner Inner; Fn func() }
type hostLayout struct{}
type Fake struct { _ hostLayout }
type Ordinary struct { _ Fake; B byte; F float64; P *byte }
type HostTail struct { _ structs.HostLayout; Value uint64; Tail struct{} }
type NativeCallback func()
type Callbacks struct { _ structs.HostLayout; Native NativeCallback; Go func() }
type CallbackArrays struct { _ structs.HostLayout; Native [2]NativeCallback; Go [2]func() }
type AliasCallback = func()
type AliasCallbacks struct { _ structs.HostLayout; Native AliasCallback; Go func() }
const Size = unsafe.Sizeof(Record{})
const Offset = unsafe.Offsetof(Record{}.U)
`, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			prog.SetTypeCallConv("host.NativeCallback", CDecl)
			prog.SetTypeCallConv("host.AliasCallback", CDecl)
			conf := types.Config{Importer: importer.Default(), Sizes: sizes}
			pkg, err := conf.Check("host", fset, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if pkg.Scope().Lookup("Size").(*types.Const).Val().ExactString() != strconv.FormatInt(tc.size, 10) || pkg.Scope().Lookup("Offset").(*types.Const).Val().ExactString() != strconv.FormatInt(tc.offsets[4], 10) {
				t.Fatal("checker constants do not use host layout")
			}
			for _, name := range []string{"Record", "Derived", "Anonymous", "Instance"} {
				raw := pkg.Scope().Lookup(name).Type()
				checkHostLayout(t, prog, sizes, raw, tc.size, tc.alignment, tc.offsets)
				checkHostLayout(t, prog, sizes, types.NewArray(raw, 2), 2*tc.size, tc.alignment, nil)
			}
			tailSize := align(tc.size+1, tc.alignment)
			if tc.arch == "arm" {
				tailSize = tc.size
			}
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("GoTail").Type(), tailSize, tc.alignment, []int64{0, tc.size})
			outerSize := align(tc.alignment+tc.size+1, tc.alignment)
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("Outer").Type(), outerSize, tc.alignment, []int64{0, tc.alignment, tc.alignment + tc.size})
			inner := pkg.Scope().Lookup("Inner").Type()
			word := int64(prog.GoWordSize())
			innerSize, innerAlign := sizes.Sizeof(inner), sizes.Alignof(inner)
			ordinary := pkg.Scope().Lookup("Ordinary").Type()
			if tc.arch == "arm" {
				// Ordinary source sizing stays unchanged, as does its preexisting
				// LLVM layout. HostLayout must use the latter for nested storage.
				if innerSize != 16 || innerAlign != 4 || sizes.Sizeof(ordinary) != 16 || sizes.Alignof(ordinary) != 4 {
					t.Fatal("HostLayout changed ordinary source sizing")
				}
				lowered := prog.Type(ordinary, GoDecl)
				if prog.SizeOf(lowered) != 24 || prog.AlignOf(lowered) != 8 || prog.OffsetOf(lowered, 2) != 8 || prog.OffsetOf(lowered, 3) != 16 {
					t.Fatal("HostLayout changed ordinary LLVM layout")
				}
				innerSize, innerAlign = 24, 8
			} else {
				checkHostLayout(t, prog, sizes, ordinary, innerSize, innerAlign, []int64{0, 0, sizes.Offsetsof(structFields(inner))[1], sizes.Offsetsof(structFields(inner))[2]})
			}
			fnOffset := align(innerSize, word)
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("Nested").Type(), align(fnOffset+2*word, max(innerAlign, word)), max(innerAlign, word), []int64{0, 0, fnOffset})
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("HostTail").Type(), align(9, tc.alignment), tc.alignment, []int64{0, 0, 8})
			fnOffset = align(int64(prog.PointerSize()), word)
			for _, name := range []string{"Callbacks", "AliasCallbacks"} {
				checkHostLayout(t, prog, sizes, pkg.Scope().Lookup(name).Type(), fnOffset+2*word, word, []int64{0, 0, fnOffset})
			}
			fnOffset = align(2*int64(prog.PointerSize()), word)
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("CallbackArrays").Type(), fnOffset+4*word, word, []int64{0, 0, fnOffset})
		})
	}
}

func structFields(t types.Type) []*types.Var {
	st := t.Underlying().(*types.Struct)
	fields := make([]*types.Var, st.NumFields())
	for i := range fields {
		fields[i] = st.Field(i)
	}
	return fields
}

func checkHostLayout(t *testing.T, prog Program, sizes types.Sizes, raw types.Type, size, alignment int64, offsets []int64) {
	t.Helper()
	if got := sizes.Sizeof(raw); got != size {
		t.Fatalf("%s source size = %d, want %d", raw, got, size)
	}
	if got := sizes.Alignof(raw); got != alignment {
		t.Fatalf("%s source align = %d, want %d", raw, got, alignment)
	}
	lowered := prog.Type(raw, GoDecl)
	if got := prog.SizeOf(lowered); got != uint64(size) {
		t.Fatalf("%s LLVM size = %d, want %d; %s", raw, got, size, lowered.ll)
	}
	if got := prog.AlignOf(lowered); got != uint64(alignment) {
		t.Fatalf("%s LLVM align = %d, want %d", raw, got, alignment)
	}
	if offsets != nil {
		if got := sizes.Offsetsof(structFields(raw)); !reflect.DeepEqual(got, offsets) {
			t.Fatalf("%s offsets = %v, want %v", raw, got, offsets)
		}
		for i, want := range offsets {
			if got := prog.OffsetOf(lowered, i); got != uint64(want) {
				t.Fatalf("%s field %d LLVM offset = %d, want %d", raw, i, got, want)
			}
		}
	}
}
