package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"math"
	"reflect"
	"testing"
)

func TestHostLayoutNestedNativeTypes(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "linux", GOARCH: "riscv32", LLVMTarget: "riscv32-unknown-elf"},
	} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(&types.StdSizes{WordSize: 4, MaxAlign: 4})
			setTestRuntime(t, prog)
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "native.go", `package native
import "structs"
import "unsafe"
type Headers struct { S string; I interface{}; Slice []byte; M map[byte]byte; C chan byte }
type Inner struct { B byte; F float64; P *byte }
type PlainOuter struct { I Inner; Tail byte }
type HostOuter struct { _ structs.HostLayout; I Inner; Tail byte }
type ArrayBox struct { Items [2]Inner; Tail byte }
type ZeroTail struct { F float64; Tail struct{} }
type FuncFirst struct { F func(); D float64 }
type Callback func()
type AliasCallback = func()
type Callbacks struct { C Callback; Alias AliasCallback; Go func(); Tail byte }
type Legacy struct { F func(); Tail byte }
type LegacyAlias = struct { F func(); Tail byte }
type LegacyArray = [2]func()
// This feature must not change standalone ordinary Go source queries.
const OrdinarySize = unsafe.Sizeof(Inner{})
const OrdinaryAlignment = unsafe.Alignof(Inner{})
const OrdinaryOffset = unsafe.Offsetof(Inner{}.F)
const Size = unsafe.Sizeof(HostOuter{})
const Alignment = unsafe.Alignof(HostOuter{})
const Offset = unsafe.Offsetof(HostOuter{}.Tail)
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			prog.SetTypeCallConv("native.Callback", CDecl)
			prog.SetTypeCallConv("native.AliasCallback", CDecl)
			prog.SetTypeCallConv("native.Legacy", CDecl)
			prog.SetTypeCallConv("native.LegacyAlias", CDecl)
			prog.SetTypeCallConv("native.LegacyArray", CDecl)
			// Offsetsof sees fields rather than the enclosing named type.
			ast.Inspect(file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok && (ts.Name.Name == "Legacy" || ts.Name.Name == "LegacyAlias") {
					var positions []token.Pos
					for _, field := range ts.Type.(*ast.StructType).Fields.List {
						positions = append(positions, field.Names[0].Pos())
					}
					prog.SetNativeStructFields(positions)
				}
				return true
			})
			pkg, err := (&types.Config{Importer: importer.Default(), Sizes: sizes}).Check("native", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]string{
				"Size": "32", "Alignment": "8", "Offset": "24",
				"OrdinarySize": "16", "OrdinaryAlignment": "4", "OrdinaryOffset": "4",
			} {
				if got := pkg.Scope().Lookup(name).(*types.Const).Val().ExactString(); got != want {
					t.Errorf("%s = %s, want %s", name, got, want)
				}
			}
			marker := pkg.Scope().Lookup("HostOuter").Type().Underlying().(*types.Struct).Field(0).Type()
			wrap := func(raw types.Type) types.Type {
				return types.NewStruct([]*types.Var{
					types.NewField(token.NoPos, nil, "_", marker, false),
					types.NewField(token.NoPos, nil, "Value", raw, false),
				}, nil)
			}
			for _, tc := range []struct {
				name        string
				size, align int64
				offsets     []int64
			}{
				{"Headers", 36, 4, []int64{0, 8, 16, 28, 32}},
				{"Inner", 24, 8, []int64{0, 8, 16}},
				{"PlainOuter", 32, 8, []int64{0, 24}},
				{"HostOuter", 32, 8, []int64{0, 0, 24}},
				{"ArrayBox", 56, 8, []int64{0, 48}},
				{"ZeroTail", 8, 8, []int64{0, 8}},
				{"FuncFirst", 16, 8, []int64{0, 8}},
				{"Callbacks", 16, 4, []int64{0, 4, 8, 12}},
				{"Legacy", 8, 4, []int64{0, 4}},
				{"LegacyAlias", 8, 4, []int64{0, 4}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := pkg.Scope().Lookup(tc.name).Type()
					// Lower the standalone type before its HostLayout container:
					// this is its existing storage, not a layout inherited from the
					// parent's marker. Ordinary source/metadata discrepancies such
					// as Inner's 16/4 versus storage 24/8 are tracked by #2810.
					lowered := prog.Type(raw, GoDecl)
					if prog.SizeOf(lowered) != uint64(tc.size) || prog.AlignOf(lowered) != uint64(tc.align) {
						t.Fatalf("%s: unexpected nested storage size/alignment", tc.name)
					}
					for i, want := range tc.offsets {
						if got := prog.OffsetOf(lowered, i); got != uint64(want) {
							t.Fatalf("nested storage field %d offset = %d, want %d", i, got, want)
						}
					}
					host := wrap(raw)
					checkHostLayout(t, prog, sizes, host, tc.size, tc.align, []int64{0, 0})
					reflected := (&aBuilder{Prog: prog}).abiStructFieldOffsets(host.Underlying().(*types.Struct))
					if !reflect.DeepEqual(reflected, []int64{0, 0}) {
						t.Fatalf("HostLayout reflected offsets = %v, want [0 0]", reflected)
					}
					checkHostLayout(t, prog, sizes, wrap(types.NewArray(raw, 2)), 2*tc.size, tc.align, []int64{0, 0})
				})
			}
			checkHostLayout(t, prog, sizes, wrap(pkg.Scope().Lookup("LegacyArray").Type()), 8, 4, []int64{0, 0})
			closure := prog.Closure(NoArgsNoRet)
			checkHostLayout(t, prog, sizes, wrap(closure.RawType()), 4, 4, []int64{0, 0})
		})
	}
}

// Oversized HostLayout aggregates must be rejected before LLVM types are constructed.
func TestHostLayoutOverflow(t *testing.T) {
	Initialize(InitAll)
	for _, arch := range []string{"arm", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: arch})
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", arch))
			pkg, err := importer.Default().Import("structs")
			if err != nil {
				t.Fatal(err)
			}
			marker := pkg.Scope().Lookup("HostLayout").Type()
			marked := types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "_", marker, false)}, nil)
			empty := types.NewStruct(nil, nil)
			huge := types.NewArray(types.Typ[types.Byte], math.MaxInt64)
			// A containing Go struct has a trailing zero-size field. On amd64 the
			// extra byte needed for that field must not wrap around MaxInt64.
			tailFields := []*types.Var{
				types.NewField(token.NoPos, nil, "Marker", marked, false),
				types.NewField(token.NoPos, nil, "Data", huge, false),
				types.NewField(token.NoPos, nil, "Tail", empty, false),
			}
			if arch == "amd64" && sizes.Sizeof(types.NewStruct(tailFields, nil)) >= 0 {
				t.Fatal("trailing padding overflow was not detected")
			}
			// A directly marked struct retains tail protection on every target.
			tailFields[0] = types.NewField(token.NoPos, nil, "_", marker, false)
			if sizes.Sizeof(types.NewStruct(tailFields, nil)) >= 0 {
				t.Fatal("HostLayout trailing padding overflow was not detected")
			}
			overflowing := types.NewArray(types.Typ[types.Int64], math.MaxInt64)
			fields := []*types.Var{
				types.NewField(token.NoPos, nil, "_", marker, false),
				types.NewField(token.NoPos, nil, "Data", overflowing, false),
				types.NewField(token.NoPos, nil, "Tail", types.Typ[types.Byte], false),
			}
			if sizes.Sizeof(types.NewStruct(fields, nil)) >= 0 || sizes.Offsetsof(fields)[2] >= 0 {
				t.Fatal("aggregate overflow was not detected")
			}
		})
	}
}
