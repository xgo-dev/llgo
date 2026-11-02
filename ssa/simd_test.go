//go:build !llgo

package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDFeaturesPreserveExistingRequirements(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "p")
	fn := pkg.NewFunc("p.f", NoArgsNoRet, GoDecl)
	fn.impl.AddFunctionAttr(prog.ctx.CreateStringAttribute("target-features", "+bulk-memory"))
	b := fn.MakeBody(1)
	b.requireSIMDFeatures()
	b.requireSIMDFeatures()
	b.Return()
	b.EndBuild()
	for _, attr := range fn.impl.GetFunctionAttributes() {
		if attr.IsString() && attr.GetStringKind() == "target-features" {
			got := attr.GetStringValue()
			if !strings.Contains(got, "+bulk-memory") || strings.Count(got, "+simd128") != 1 {
				t.Fatalf("target features = %q", got)
			}
			return
		}
	}
	t.Fatal("missing target features")
}

func TestSIMDValueAndStorageLayout(t *testing.T) {
	rawPkg := types.NewPackage("simd/archsimd", "archsimd")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "v128", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, rawPkg, "tag", types.NewArray(types.NewSignatureType(nil, nil, nil, nil, nil, false), 0), false),
	}, nil), nil)
	for _, target := range []*Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"}, {GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			raw := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "Float32x4", nil), types.NewStruct([]*types.Var{
				types.NewField(token.NoPos, rawPkg, "tag", marker, false),
				types.NewField(token.NoPos, rawPkg, "vals", types.NewArray(types.Typ[types.Float32], 4), false),
			}, nil), nil)
			sizes := prog.TypeSizes(&types.StdSizes{WordSize: int64(prog.GoWordSize()), MaxAlign: int64(prog.GoWordSize())})
			// Query source types before any importer, native metadata
			// registration, or LLVM type construction has run.
			defined := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("p", "p"), "V", nil), raw.Underlying(), nil)
			zeroTail := types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "x", types.Typ[types.Byte], false), types.NewField(token.NoPos, nil, "z", types.NewArray(defined, 0), false)}, nil)
			if sizes.Alignof(defined) != 16 || sizes.Alignof(zeroTail) != 16 || sizes.Sizeof(zeroTail) != 32 {
				t.Fatal("source SIMD layout before LLVM construction")
			}
			if sizes.Offsetsof([]*types.Var{zeroTail.Field(0), zeroTail.Field(1)})[1] != 16 {
				t.Fatal("source SIMD field offset before LLVM construction")
			}
			if err := prog.ValidateVectorMaps(types.NewMap(types.Typ[types.Int], defined)); err == nil {
				t.Fatal("source SIMD map storage accepted before LLVM construction")
			}
			typ := prog.rawType(raw)
			if typ.ll.TypeKind() != llvm.VectorTypeKind || prog.SizeOf(typ) != 16 || prog.AlignOf(typ) != 16 || prog.OffsetOf(typ, 1) != 0 {
				t.Fatal("incorrect vector value or Go storage layout")
			}
			fields := types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "prefix", types.Typ[types.Byte], false), types.NewField(token.NoPos, nil, "value", raw, false), types.NewField(token.NoPos, nil, "suffix", types.Typ[types.Byte], false)}, nil)
			record := prog.rawType(fields)
			if prog.OffsetOf(record, 1) != 16 || prog.OffsetOf(record, 2) != 32 || prog.SizeOf(record) != 48 {
				t.Fatal("native vector alignment missing from Go struct layout")
			}
			tailType := prog.rawType(zeroTail)
			if prog.SizeOf(tailType) != 32 || prog.AlignOf(tailType) != 16 || prog.OffsetOf(tailType, 1) != 16 {
				t.Fatal("zero-sized vector tail LLVM layout", tailType.ll.String())
			}
			if err := prog.ValidateVectorMaps(types.NewMap(types.Typ[types.Int], defined)); err == nil {
				t.Fatal("defined SIMD map value accepted")
			}
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("p", "p")
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "x", raw)), types.NewTuple(types.NewParam(token.NoPos, nil, "", raw)), false)
			fn := pkg.NewFunc("p.roundtrip", sig, GoDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			ptr := b.AllocaT(typ)
			// Source-level field access and aggregate construction must bridge the
			// representation used inside archsimd without changing its Go type.
			rebuilt := b.Aggregate(typ, prog.Zero(prog.rawType(marker)), b.Field(fn.Param(0), 1))
			converted := b.ChangeType(typ, b.ChangeType(prog.rawType(raw.Underlying()), rebuilt))
			if converted.Type != typ {
				t.Fatal("SIMD conversion lost its named type")
			}
			b.Store(ptr, fn.Param(0))
			b.Return(b.Load(ptr))
			b.EndBuild()
			ir := fn.impl.String()
			if !strings.Contains(ir, "load <4 x float>") || !strings.Contains(ir, "store <4 x float>") {
				t.Fatalf("memory access scalarizes vectors:\n%s", ir)
			}
			clearSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "p", types.NewPointer(raw))), nil, false)
			clear := pkg.NewFunc("p.clear", clearSig, GoDecl)
			cb := clear.MakeBody(1)
			cb.Store(clear.Param(0), prog.Zero(typ))
			cb.Return()
			cb.EndBuild()
			cb.Dispose()
			zero := prog.Zero(typ)
			constant := prog.toStorageConstant(typ, zero.impl)
			if constant.Type() != prog.storageType(typ) {
				t.Fatal("wrong global storage constant")
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSIMDTagNativeAlignment(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64", "wasm"} {
		t.Run(arch, func(t *testing.T) {
			target := &Target{GOOS: "linux", GOARCH: arch}
			if arch == "wasm" {
				target.GOOS = "wasip1"
				target.WasmProfile = "w32"
				target.WasmProvider = "wasi"
			}
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(&types.StdSizes{WordSize: int64(prog.GoWordSize()), MaxAlign: int64(prog.GoWordSize())})
			pkg := types.NewPackage("simd/archsimd", "archsimd")
			for _, width := range []int{16, 32, 64} {
				tag := types.NewNamed(types.NewTypeName(token.NoPos, pkg, fmt.Sprintf("v%d", width*8), nil), types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "tag", types.NewArray(types.Typ[types.Uint8], 0), false)}, nil), nil)
				want := width
				if arch == "arm64" {
					want = 16
				}
				if sizes.Sizeof(tag) != 0 || sizes.Alignof(tag) != int64(want) {
					t.Fatalf("source tag v%d: size/align %d/%d", width*8, sizes.Sizeof(tag), sizes.Alignof(tag))
				}
				typ := prog.rawType(tag)
				if prog.SizeOf(typ) != 0 || prog.AlignOf(typ) != uint64(want) {
					t.Fatalf("LLVM tag v%d: size/align %d/%d", width*8, prog.SizeOf(typ), prog.AlignOf(typ))
				}
			}
		})
	}
}
