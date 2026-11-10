package ssa

import (
	"bytes"
	"debug/pe"
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestGoBasicAccessAlignment(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []Target{
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+target.GOARM, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			sizes := types.SizesFor("gc", target.GOARCH)
			prog.TypeSizes(sizes)
			pkg := prog.NewPackage("example.com/alignment", "alignment")
			for _, basic := range []types.BasicKind{types.Byte, types.Int32, types.Int64, types.Float32, types.Float64, types.Complex64, types.Complex128} {
				raw := types.Typ[basic]
				for _, field := range []bool{false, true} {
					name := raw.Name() + "_pointer"
					var elem types.Type = raw
					if field {
						name = raw.Name() + "_field"
						elem = types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "B", types.Typ[types.Byte], false), types.NewField(token.NoPos, nil, "V", raw, false)}, nil)
					}
					t.Run(name, func(t *testing.T) {
						sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewPointer(elem))), types.NewTuple(types.NewVar(token.NoPos, nil, "", raw)), false)
						fn := pkg.NewFunc(name, sig, CDecl)
						b := fn.MakeBody(1)
						ptr := fn.Param(0)
						if field {
							ptr = b.FieldAddr(ptr, 1)
						}
						typ := prog.Elem(ptr.Type)
						stored := b.Store(ptr, prog.Zero(typ))
						loaded := b.Load(ptr)
						b.Return(loaded)
						b.EndBuild()
						want := int(sizes.Alignof(raw))
						for name, inst := range map[string]llvm.Value{"load": loaded.impl, "store": stored.impl} {
							if got := inst.Alignment(); got != want {
								t.Errorf("%s alignment = %d, want Go pointer guarantee %d", name, got, want)
							}
						}
						b.Dispose()
					})
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			obj, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			obj.Dispose()
		})
	}
}

// Atomic operations require natural alignment even when ordinary Go pointers
// have a weaker guarantee. In particular, underaligned i64 operations on
// Windows/386 lower to generic libatomic helpers that the runtime does not link.
func TestGoAtomicAccessAlignment(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []Target{
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+target.GOARM, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			sizes := types.SizesFor("gc", target.GOARCH)
			prog.TypeSizes(sizes)
			pkg := prog.NewPackage("example.com/atomic_alignment", "atomic_alignment")
			for _, basic := range []types.BasicKind{types.Uint8, types.Int32, types.Int64, types.Uint64, types.Uintptr, types.UnsafePointer} {
				raw := types.Typ[basic]
				for _, ordering := range []AtomicOrdering{OrderingNotAtomic, OrderingUnordered, OrderingSeqConsistent} {
					name := fmt.Sprintf("%s_%d", raw.Name(), ordering)
					t.Run(name, func(t *testing.T) {
						sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewPointer(raw)), types.NewVar(token.NoPos, nil, "v", raw)), types.NewTuple(types.NewVar(token.NoPos, nil, "", raw)), false)
						fn := pkg.NewFunc(name, sig, CDecl)
						b := fn.MakeBody(1)
						defer b.Dispose()
						stored := b.Store(fn.Param(0), fn.Param(1)).SetOrdering(ordering)
						loaded := b.Load(fn.Param(0)).SetOrdering(ordering)
						b.Return(loaded)
						b.EndBuild()
						want := int(sizes.Alignof(raw))
						if ordering != OrderingNotAtomic && (basic == types.Int64 || basic == types.Uint64) {
							want = 8
						}
						for name, expr := range map[string]Expr{"load": loaded, "store": stored} {
							inst := expr.memoryInstruction()
							if got := inst.Alignment(); got != want {
								t.Errorf("%s alignment = %d, want %d", name, got, want)
							}
							if got := inst.Ordering(); got != ordering {
								t.Errorf("%s ordering = %v, want %v", name, got, ordering)
							}
						}
					})
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			obj, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			defer obj.Dispose()
			if target.GOOS == "windows" {
				coff, err := pe.NewFile(bytes.NewReader(obj.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				defer coff.Close()
				for _, sym := range coff.Symbols {
					if sym.SectionNumber == 0 && (strings.Contains(sym.Name, "__atomic_load") || strings.Contains(sym.Name, "__atomic_store")) {
						t.Errorf("unexpected atomic helper reference: %s", sym.Name)
					}
				}
			}
		})
	}
}
