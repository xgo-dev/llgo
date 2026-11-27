package cl

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/abi"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestConvertedLargeStructFieldDoesNotAllocate(t *testing.T) {
	const src = `package projection
type Inner struct { Data [70000]byte; Tag uint64 }
type Outer struct { Prefix uint64; Inner Inner }
type Converted Outer
func Read(v *Outer) uint64 { return Converted(*v).Inner.Tag }
`
	pkg, _, files := buildGoSSAPkg(t, src)
	for _, target := range []*llvmgen.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
	} {
		t.Run(target.GOOS, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, target)
			defer prog.Dispose()
			lp, err := NewPackage(prog, pkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := lp.Module()
			cfg := abi.AggregateLoweringConfig{GoWordSize: prog.GoWordSize(), Wasm: target.GOARCH == "wasm"}
			abi.LowerLargeAggregates(prog.TargetData(), mod, cfg)
			abi.LowerAggregateCopies(prog.TargetData(), mod, cfg)
			if ir := mustNamedFunction(t, mod, "projection.Read").String(); strings.Contains(ir, "runtime.Alloc") {
				t.Fatalf("reading a scalar field allocated an aggregate snapshot:\n%s", ir)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNestedLargeResultKeepsPackedFieldAlignment(t *testing.T) {
	const src = `package projection
//llgo:cdecl
type Item struct { N uint64; Data [8193]uint64 }
type Mid struct { Item Item }
type Outer struct { X uint32; Mid Mid }
func result() Outer
func Read() uint64 { return result().Mid.Item.N }
`
	pkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llvmgen.Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	lp, err := NewPackage(prog, pkg, files)
	if err != nil {
		t.Fatal(err)
	}
	mod := lp.Module()
	td := prog.TargetData()
	outer := mustNamedFunction(t, mod, "projection.result").GlobalValueType().ReturnType()
	if offset := td.ElementOffset(outer, 1); offset != 4 {
		t.Fatalf("nested record starts at offset %d, want 4", offset)
	}
	abi.LowerLargeAggregates(td, mod, abi.AggregateLoweringConfig{GoWordSize: prog.GoWordSize()})
	fn := mustNamedFunction(t, mod, "projection.Read")
	load := fn.FirstBasicBlock().LastInstruction().Operand(0).IsALoadInst()
	if load.IsNil() || load.Alignment() > 4 {
		t.Fatalf("expected directly projected field with at most 4-byte alignment:\n%s", fn)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
