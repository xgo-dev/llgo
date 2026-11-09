package ssa

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

type alignmentImporter struct {
	types.Importer
	layout *types.Package
}

func (i alignmentImporter) Import(path string) (*types.Package, error) {
	if path == i.layout.Path() {
		return i.layout, nil
	}
	return i.Importer.Import(path)
}

func alignmentPackage(t *testing.T, sizes types.Sizes, source string) (*types.Package, *types.Package) {
	t.Helper()
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "../runtime/_patch/structs/alignment_llgo.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	host, err := parser.ParseFile(fs, filepath.Join(runtime.GOROOT(), "src", "structs", "hostlayout.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := (&types.Config{Sizes: sizes}).Check("structs", fs, []*ast.File{host, file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	file, err = parser.ParseFile(fs, "alignment.go", "package alignment\nimport \"structs\"\n"+source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Sizes: sizes, Importer: alignmentImporter{importer.Default(), layout}}).Check("alignment", fs, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg, layout
}

func TestExplicitAlignmentLayout(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"}, {GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm"}, {GOOS: "linux", GOARCH: "riscv64"},
		{GOOS: "linux", GOARCH: "riscv32", LLVMTarget: "riscv32-unknown-elf"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			base := types.SizesFor("gc", target.GOARCH)
			if base == nil {
				base = &types.StdSizes{WordSize: int64(prog.GoWordSize()), MaxAlign: int64(prog.GoWordSize())}
			}
			sizes := prog.TypeSizes(base)
			pkg, layout := alignmentPackage(t, sizes, `
import "unsafe"
type Block struct { _ structs.Align16; Data [24]byte }
type Derived Block
type Alias = Block
type Marker structs.Align32
type Marked struct { _ Marker; B byte }
type Embedded struct { structs.Align64; B byte }
type Generic[T any] struct { _ structs.Align16; Value T }
type Instance = Generic[[24]byte]
type Outer struct { B byte; Value Block; Tail byte }
type Tail struct { B byte; _ structs.Align16 }
type Middle struct { B byte; _ structs.Align16; V byte }
type Zero struct { _ structs.Align64 }
type Multiple struct { _ structs.Align8; _ structs.Align64; _ structs.Align16; B byte }
type Host struct { _ structs.HostLayout; _ structs.Align64; B byte; F float64 }
type Inner struct { B byte; F float64; Tail byte }
type NestedOrdinary struct { _ structs.Align16; Inner Inner; Tail byte }
type NestedHost struct { B byte; Inner Host; Tail byte }
type NestedArray struct { _ structs.Align16; Inner [2]Inner; Tail byte }
type AtomicLike struct { _ structs.Align8; B uint32; V uint64 }
type AlignedUint64 struct { _ structs.Align8; V uint64 }
type AtomicField struct { B uint32; V AlignedUint64 }
type align64 struct{}
type Fake struct { _ align64; B byte }
const Size = unsafe.Sizeof(Block{})
const Alignment = unsafe.Alignof(Block{})
const Offset = unsafe.Offsetof(Outer{}.Value)
`)
			for name, want := range map[string]string{"Size": "32", "Alignment": "16", "Offset": "16"} {
				if got := pkg.Scope().Lookup(name).(*types.Const).Val().ExactString(); got != want {
					t.Fatalf("constant %s = %s, want %s", name, got, want)
				}
			}
			for _, n := range []int64{8, 16, 32, 64} {
				raw := layout.Scope().Lookup(fmt.Sprintf("Align%d", n)).Type()
				checkHostLayout(t, prog, sizes, raw, 0, n, []int64{0})
				if !types.Comparable(raw) {
					t.Fatal("alignment marker is not comparable")
				}
			}
			innerRaw := pkg.Scope().Lookup("Inner").Type()
			innerSize := int64(prog.SizeOf(prog.Type(innerRaw, GoDecl)))
			for _, tc := range []struct {
				name        string
				size, align int64
				offsets     []int64
			}{
				{"Block", 32, 16, []int64{0, 0}}, {"Derived", 32, 16, []int64{0, 0}},
				{"Alias", 32, 16, []int64{0, 0}}, {"Instance", 32, 16, []int64{0, 0}},
				{"Marked", 32, 32, []int64{0, 0}}, {"Embedded", 64, 64, []int64{0, 0}},
				{"Outer", 64, 16, []int64{0, 16, 48}}, {"Tail", 32, 16, []int64{0, 16}},
				{"Middle", 32, 16, []int64{0, 16, 16}}, {"Zero", 0, 64, []int64{0}},
				{"Multiple", 64, 64, []int64{0, 0, 0, 0}}, {"Fake", 1, 1, []int64{0, 0}},
				{"Host", 64, 64, []int64{0, 0, 0, prog.hostBasicLayouts[types.Float64].align}},
				{"AtomicLike", 16, 8, []int64{0, 0, sizes.Alignof(types.Typ[types.Uint64])}},
				{"AtomicField", 16, 8, []int64{0, 8}},
				{"NestedOrdinary", align(innerSize+1, 16), 16, []int64{0, 0, innerSize}},
				{"NestedArray", align(2*innerSize+1, 16), 16, []int64{0, 0, 2 * innerSize}},
				{"NestedHost", 192, 64, []int64{0, 64, 128}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := pkg.Scope().Lookup(tc.name).Type()
					checkHostLayout(t, prog, sizes, raw, tc.size, tc.align, tc.offsets)
					for _, n := range []int64{0, 3} {
						checkHostLayout(t, prog, sizes, types.NewArray(raw, n), n*tc.size, tc.align, nil)
					}
					if !types.Comparable(raw) || prog.Type(raw, GoDecl).kind != vkStruct {
						t.Fatal("alignment changed comparability or value representation")
					}
				})
			}
			// Check native object emission as well as layout numbers. Keep an
			// addressable local so the backend must honor stack realignment.
			mod := prog.NewPackage("alignment.object", "object").Module()
			raw := pkg.Scope().Lookup("Embedded").Type()
			ll := prog.Type(raw, GoDecl).ll
			global := llvm.AddGlobal(mod, ll, "aligned_global")
			global.SetInitializer(llvm.ConstNull(ll))
			prog.requireStorageAlignment(global, prog.Type(raw, GoDecl))
			ptr := llvm.PointerType(prog.tyInt8(), 0)
			consumeTy := llvm.FunctionType(prog.ctx.VoidType(), []llvm.Type{ptr}, false)
			consume := llvm.AddFunction(mod, "consume", consumeTy)
			fn := llvm.AddFunction(mod, "aligned_stack", llvm.FunctionType(prog.ctx.VoidType(), nil, false))
			b := prog.ctx.NewBuilder()
			b.SetInsertPointAtEnd(prog.ctx.AddBasicBlock(fn, "entry"))
			local := llvm.CreateAlloca(b, ll)
			prog.requireStorageAlignment(local, prog.Type(raw, GoDecl))
			if prog.TargetData().ABITypeAlignment(ll) < 64 && (global.Alignment() != 64 || local.Alignment() != 64) {
				t.Fatal("missing explicit storage alignment")
			}
			b.CreateCall(consumeTy, consume, []llvm.Value{local}, "")
			b.CreateRetVoid()
			b.Dispose()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			object, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			object.Dispose()
		})
	}
}

func TestExplicitAlignmentBoundaries(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	sizes := prog.TypeSizes(types.SizesFor("gc", "amd64"))
	pkg, _ := alignmentPackage(t, sizes, `type Block struct { _ structs.Align64; Data [24]byte }; type Small struct { _ structs.Align8; V uint64 }`)
	block := pkg.Scope().Lookup("Block").Type()
	small := pkg.Scope().Lookup("Small").Type()
	for _, typ := range []types.Type{block, types.NewArray(block, 0), types.NewArray(block, 2)} {
		for _, m := range []*types.Map{types.NewMap(typ, types.Typ[types.Int]), types.NewMap(types.Typ[types.Int], typ)} {
			if err := prog.ValidateVectorMaps(m); err == nil || !strings.Contains(err.Error(), "explicit alignment above 8 bytes") {
				t.Fatalf("missing map diagnostic: %v", err)
			}
		}
	}
	for _, typ := range []types.Type{small, types.NewPointer(block), types.NewSlice(block), types.NewChan(types.SendRecv, block)} {
		if err := prog.ValidateVectorMaps(types.NewMap(types.Typ[types.Int], typ)); err != nil {
			t.Fatal(err)
		}
	}
	for _, typ := range []types.Type{small, block, types.NewArray(block, 0)} {
		for _, result := range []bool{false, true} {
			func() {
				defer func() {
					if v := recover(); v == nil || !strings.Contains(fmt.Sprint(v), "explicitly aligned values at native C call boundaries") {
						t.Fatalf("missing native boundary diagnostic: %v", v)
					}
				}()
				tuple := types.NewTuple(types.NewParam(token.NoPos, nil, "", typ))
				params, results := tuple, (*types.Tuple)(nil)
				if result {
					params, results = nil, tuple
				}
				prog.FuncDecl(types.NewSignatureType(nil, nil, nil, params, results, false), CDecl)
			}()
		}
	}
	tuple := types.NewTuple(types.NewParam(token.NoPos, nil, "", types.NewPointer(block)))
	prog.FuncDecl(types.NewSignatureType(nil, nil, nil, tuple, tuple, false), CDecl)
}

func TestExplicitAlignmentDebugMetadata(t *testing.T) {
	for _, arch := range []string{"amd64", "arm"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: arch})
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", arch))
			rawPkg, _ := alignmentPackage(t, sizes, `type Block struct { _ structs.Align64; B byte }; type Outer struct { B byte; Value Block }`)
			pkg := prog.NewPackage("alignment", "alignment")
			pkg.InitDebug("alignment", "alignment", token.NewFileSet())
			raw := rawPkg.Scope().Lookup("Outer").Type()
			global := pkg.NewVar("alignment.global", types.NewPointer(raw), GoDecl)
			global.Init(prog.Zero(prog.Type(raw, GoDecl)))
			fn := pkg.NewFunc("alignment.debug", NoArgsNoRet, GoDecl)
			b := fn.MakeBody(1)
			b.DIGlobal(global.Expr, "alignment.global", token.Position{Filename: "alignment.go", Line: 1})
			b.Return()
			b.EndBuild()
			pkg.FinalizeDebug()
			foundType, foundField := false, false
			for _, line := range strings.Split(pkg.String(), "\n") {
				if strings.Contains(line, "DW_TAG_typedef") && strings.Contains(line, "Outer") && strings.Contains(line, "align: 512") {
					foundType = true
				}
				if strings.Contains(line, `name: "Value"`) && strings.Contains(line, "offset: 512") && strings.Contains(line, "align: 512") {
					foundField = true
				}
			}
			if !foundType || !foundField {
				t.Fatalf("missing explicit alignment in DWARF:\n%s", pkg.String())
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
