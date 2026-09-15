package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

func wasmAggregateConfig(bits int, roots bool) abi.AggregateLoweringConfig {
	return abi.AggregateLoweringConfig{GoWordSize: bits / 8, GCRoots: roots, Wasm: true}
}

func parseWasmAggregateIR(t *testing.T, source string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "aggregate.ll")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mod.Dispose)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	return mod
}

func TestLowerWasmAggregateCopies(t *testing.T) {
	for _, bits := range []int{32, 64} {
		t.Run(fmt.Sprint(bits), func(t *testing.T) {
			td := llvm.NewTargetData(fmt.Sprintf("e-p:%d:%d-i64:64-n32:64-S128", bits, bits))
			defer td.Dispose()
			for _, tc := range []struct {
				name, body, intrinsic string
				volatile              bool
			}{
				{"copy", "%v = load [8192 x i8], ptr %src\nstore [8192 x i8] %v, ptr %dst", "memmove", false},
				{"volatile source", "%v = load volatile [8192 x i8], ptr %src\nstore [8192 x i8] %v, ptr %dst", "memmove", true},
				{"volatile destination", "%v = load [8192 x i8], ptr %src\nstore volatile [8192 x i8] %v, ptr %dst", "memmove", true},
				{"overlap", "%v = load volatile [8192 x i8], ptr %src\nstore volatile [8192 x i8] %v, ptr %src", "memmove", true},
				{"zero", "store [8192 x i8] zeroinitializer, ptr %dst", "memset", false},
				{"volatile zero", "store volatile [8192 x i8] zeroinitializer, ptr %dst", "memset", true},
				{"pointer fields", "%v = load {ptr, [8192 x i8]}, ptr %src\nstore {ptr, [8192 x i8]} %v, ptr %dst", "memmove", false},
				{"threshold", "%v = load [4096 x i8], ptr %src\nstore [4096 x i8] %v, ptr %dst", "memmove", false},
				{"small", "%v = load [4095 x i8], ptr %src\nstore [4095 x i8] %v, ptr %dst", "", false},
				{"scalar", "%v = load i64, ptr %src\nstore i64 %v, ptr %dst", "", false},
				{"unsupported use", "%v = load [8192 x i8], ptr %src\n%w = insertvalue [8192 x i8] %v, i8 1, 0\nstore [8192 x i8] %w, ptr %dst", "", false},
				{"unknown value", "store [8192 x i8] poison, ptr %dst", "", false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					mod := parseWasmAggregateIR(t, "declare void @other()\ndefine void @copy(ptr %dst, ptr %src) {\n"+tc.body+"\nret void\n}")
					before := mod.String()
					want := 0
					if tc.intrinsic != "" {
						want = 1
					}
					if got := lowerWasmAggregateCopies("wasm", td, mod, wasmAggregateConfig(bits, true)); got != want {
						t.Fatalf("lowered %d operations, want %d:\n%s", got, want, mod.String())
					}
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					if want == 0 {
						if mod.String() != before {
							t.Fatal("changed an ineligible memory operation")
						}
						return
					}
					body := mod.NamedFunction("copy").String()
					intrinsic := fmt.Sprintf("@llvm.%s.p0.", tc.intrinsic)
					if !strings.Contains(body, intrinsic) || !strings.Contains(body, fmt.Sprintf("i1 %t", tc.volatile)) {
						t.Fatalf("copy lost its intrinsic or volatility:\n%s", body)
					}
					if strings.Contains(body, "alloca ") || strings.Contains(body, "AllocU") || strings.Contains(body, "load ") || strings.Contains(body, "store ") {
						t.Fatalf("copy allocated storage or retained aggregate accesses:\n%s", body)
					}
					if got := lowerWasmAggregateCopies("wasm", td, mod, wasmAggregateConfig(bits, true)); got != 0 {
						t.Fatal("copy lowering is not idempotent")
					}
				})
			}
			mod := parseWasmAggregateIR(t, "define void @copy(ptr %p) { store volatile [8192 x i8] zeroinitializer, ptr %p\nret void }")
			before := mod.String()
			if lowerWasmAggregateCopies("amd64", td, mod, wasmAggregateConfig(bits, true)) != 0 || mod.String() != before {
				t.Fatal("changed native aggregate operations")
			}
		})
	}
}

func TestLowerWasmAggregateCopiesSnapshots(t *testing.T) {
	const source = `
%WithPointer = type { ptr, [8192 x i8] }
declare void @mutate(ptr)
define ptr @copy(ptr %src, ptr %dst, ptr %other, i1 %choice) {
entry:
  %value = load volatile %WithPointer, ptr %src
  call void @mutate(ptr %src)
  %pointer = extractvalue %WithPointer %value, 0
  br i1 %choice, label %first, label %second
first:
  store %WithPointer %value, ptr %dst
  ret ptr %pointer
second:
  store %WithPointer %value, ptr %other
  ret ptr %pointer
}
define [8192 x i8] @identity([8192 x i8] %value) {
  ret [8192 x i8] %value
}
`
	for _, bits := range []int{32, 64} {
		for _, roots := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/roots=%t", bits, roots), func(t *testing.T) {
				td := llvm.NewTargetData(fmt.Sprintf("e-p:%d:%d-i64:64-n32:64-S128", bits, bits))
				defer td.Dispose()
				mod := parseWasmAggregateIR(t, source)
				identity := mod.NamedFunction("identity").String()
				if got := lowerWasmAggregateCopies("wasm", td, mod, wasmAggregateConfig(bits, roots)); got != 1 {
					t.Fatalf("lowered %d copies, want one snapshot", got)
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				fn := mod.NamedFunction("copy")
				body := fn.String()
				if strings.Count(body, "@llvm.memcpy") != 3 || strings.Count(body, "i1 true)") != 1 ||
					strings.Contains(body, "load volatile") || strings.Contains(body, "extractvalue") ||
					strings.Index(body, "@llvm.memcpy") >= strings.Index(body, "@mutate") {
					t.Fatalf("snapshot lost ordering, volatility, or branch copies:\n%s", body)
				}
				if mod.NamedFunction("identity").String() != identity {
					t.Fatal("copy lowering changed the aggregate return ABI")
				}
				if strings.Contains(body, "llvm_gc_root_chain") != roots {
					t.Fatalf("root publication does not match the GC policy:\n%s", body)
				}
				allocations := 0
				for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
					for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
						if instr.IsACallInst().IsNil() || !strings.HasSuffix(instr.CalledValue().Name(), ".AllocU") {
							continue
						}
						allocations++
						if roots {
							before, after := llvm.PrevInstruction(instr), llvm.NextInstruction(instr)
							if before.IsAStoreInst().IsNil() || before.Operand(0) != fn.Param(0) ||
								after.IsAStoreInst().IsNil() || after.Operand(0) != instr {
								t.Fatalf("source/result not published around the new allocation:\n%s", body)
							}
						}
					}
				}
				if allocations != 1 || lowerWasmAggregateCopies("wasm", td, mod, wasmAggregateConfig(bits, roots)) != 0 {
					t.Fatal("snapshot allocation was duplicated")
				}
			})
		}
	}
}

func TestLowerWasmAggregateCopiesNestedSnapshots(t *testing.T) {
	const source = `
%Deferred = type { ptr, { i32, [8192 x i8] } }
declare void @mutate(ptr)
define void @copy(ptr %src, ptr %dst, ptr %other, i1 %choice) {
entry:
  %closure = load %Deferred, ptr %src
  call void @mutate(ptr %src)
  %argument = extractvalue %Deferred %closure, 1, 1
  br i1 %choice, label %first, label %second
first:
  store [8192 x i8] %argument, ptr %dst
  ret void
second:
  store [8192 x i8] %argument, ptr %other
  ret void
}
`
	for _, bits := range []int{32, 64} {
		for _, roots := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/roots=%t", bits, roots), func(t *testing.T) {
				td := llvm.NewTargetData(fmt.Sprintf("e-p:%d:%d-i64:64-n32:64-S128", bits, bits))
				defer td.Dispose()
				mod := parseWasmAggregateIR(t, source)
				if got := lowerWasmAggregateCopies("wasm", td, mod, wasmAggregateConfig(bits, roots)); got != 2 {
					t.Fatalf("lowered %d copies, want the closure and its array argument", got)
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				body := mod.NamedFunction("copy").String()
				if strings.Contains(body, "load [8192 x i8]") || strings.Contains(body, "store [8192 x i8]") ||
					strings.Contains(body, "extractvalue") || strings.Count(body, "@llvm.memcpy") != 4 {
					t.Fatalf("nested array escaped copy lowering:\n%s", body)
				}
				if strings.Contains(body, "llvm_gc_root_chain") != roots {
					t.Fatalf("nested snapshot root publication does not match GC policy:\n%s", body)
				}
				if got := lowerWasmAggregateCopies("wasm", td, mod, wasmAggregateConfig(bits, roots)); got != 0 {
					t.Fatalf("another pass still lowered %d copies", got)
				}
			})
		}
	}
}
