package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

func TestWasmStackAddressesCodegen(t *testing.T) {
	const rootCount = 32
	for _, bits := range []int{32, 64} {
		t.Run(fmt.Sprint(bits), func(t *testing.T) {
			var counts [2]int
			for mode := range counts {
				mod := parseWasmAggregateIR(t, `
declare i32 @setjmp(ptr) returns_twice
declare void @consume(ptr)
declare i1 @again()
define i32 @roots(ptr %data, ptr %jmp, i1 %register) null_pointer_is_valid optsize {
entry:
  call void @consume(ptr %data)
  br i1 %register, label %setup, label %body
setup:
  %saved = call i32 @setjmp(ptr %jmp)
  br label %body
body:
  %result = phi i32 [0, %entry], [%saved, %setup], [%result, %body]
  %repeat = call i1 @again()
  br i1 %repeat, label %body, label %end
end:
  ret i32 %result
}
`)
				mod.SetTarget(fmt.Sprintf("wasm%d-unknown-unknown", bits))
				mod.SetDataLayout(fmt.Sprintf("e-p:%d:%d-i64:64-n32:64-S128", bits, bits))
				fn := mod.NamedFunction("roots")
				loop := fn.BasicBlocks()[2]
				frame := llabi.NewGCRootFrame(mod, fn, rootCount, bits/8, bits/8, true)
				b := mod.Context().NewBuilder()
				b.SetInsertPointBefore(llvm.NextInstruction(loop.FirstInstruction()))
				callee := mod.NamedFunction("consume")
				for _, slot := range frame.Slots {
					b.CreateStore(fn.Param(0), slot)
					llvm.CreateCall(b, callee.GlobalValueType(), callee, []llvm.Value{fn.Param(0)})
				}
				b.Dispose()
				llabi.PopGCRootFrame(mod, fn, frame)
				options := llvm.NewPassBuilderOptions()
				defer options.Dispose()
				if err := mod.RunPasses("default<Os>", llvm.TargetMachine{}, options); err != nil {
					t.Fatal(err)
				}
				if mode != 0 {
					if moved := localizeWasmStackAddresses("wasm", mod); moved != rootCount {
						t.Fatalf("localized %d stores, want %d", moved, rootCount)
					}
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				path := filepath.Join(dir, "roots.ll")
				if err := os.WriteFile(path, []byte(mod.String()), 0600); err != nil {
					t.Fatal(err)
				}
				asm := filepath.Join(dir, "roots.s")
				cmd := exec.Command("clang", fmt.Sprintf("--target=wasm%d-unknown-unknown", bits),
					"-Os", "-S", "-fwasm-exceptions", "-mllvm", "-wasm-enable-sjlj", path, "-o", asm)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("clang: %v\n%s", err, output)
				}
				output, err := os.ReadFile(asm)
				if err != nil {
					t.Fatal(err)
				}
				counts[mode] = strings.Count(string(output), fmt.Sprintf("i%d.store", bits))
			}
			t.Logf("stores before=%d after=%d", counts[0], counts[1])
			// The LLVM 22 baseline has 1124 stores: one address spill per
			// root per call. Keep the fixed case linear, without failing if a
			// later LLVM also learns to optimize the unmodified baseline.
			if counts[1] > 4*rootCount {
				t.Fatalf("quadratic SjLj stack-address spilling: %v", counts)
			}
		})
	}
}

func TestWasmStackAddressesPreserveOperations(t *testing.T) {
	const source = `
@escape = global ptr null
@jmpfn = global ptr @setjmp
declare i32 @setjmp(ptr) returns_twice
declare void @consume(ptr)
define ptr @f(ptr %data, ptr %jmp, i32 %index, i1 %choice) {
entry:
  %frame = alloca [8 x ptr], align 16
  %slot = getelementptr [8 x ptr], ptr %frame, i32 0, i32 1
  %retained = getelementptr [8 x ptr], ptr %frame, i32 0, i32 2
  %dynamic = getelementptr [8 x ptr], ptr %frame, i32 0, i32 %index
  %external = getelementptr i8, ptr %data, i32 8
  %unused = getelementptr [8 x ptr], ptr %frame, i32 0, i32 3
  store ptr %retained, ptr @escape
  br label %body
body:
  %saved = call i32 @setjmp(ptr %jmp)
  %again = call i32 @setjmp(ptr %jmp)
  %value = load ptr, ptr %retained
  store volatile ptr %data, ptr %slot, align 8
  store atomic ptr %data, ptr %retained release, align 8
  store ptr %data, ptr %dynamic
  store ptr %data, ptr %external
  br i1 %choice, label %other, label %end
other:
  store ptr null, ptr %slot
  br label %end
end:
  ret ptr %value
}
define void @ordinary(ptr %data) {
  %frame = alloca [8 x ptr]
  %slot = getelementptr [8 x ptr], ptr %frame, i32 0, i32 1
  call void @consume(ptr %frame)
  store ptr %data, ptr %slot
  ret void
}
`
	mod := parseWasmAggregateIR(t, source)
	before := mod.String()
	if got := localizeWasmStackAddresses("amd64", mod); got != 0 || mod.String() != before {
		t.Fatal("changed native code generation")
	}
	ordinary := mod.NamedFunction("ordinary").String()
	fn := mod.NamedFunction("f")
	type operation struct {
		store, value, address llvm.Value
		name                  string
		alignment             int
		volatile              bool
		ordering              llvm.AtomicOrdering
	}
	var operations []operation
	for _, bb := range fn.BasicBlocks() {
		for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if !instr.IsAStoreInst().IsNil() {
				operations = append(operations, operation{instr, instr.Operand(0), instr.Operand(1), instr.Operand(1).Name(), instr.Alignment(), instr.IsVolatile(), instr.Ordering()})
			}
		}
	}
	if moved := localizeWasmStackAddresses("wasm", mod); moved != 3 {
		t.Fatalf("localized %d stores, want 3", moved)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	if mod.NamedFunction("ordinary").String() != ordinary {
		t.Fatal("changed a function without setjmp")
	}
	for _, op := range operations {
		if op.store.Operand(0) != op.value || op.store.Alignment() != op.alignment ||
			op.store.IsVolatile() != op.volatile || op.store.Ordering() != op.ordering {
			t.Fatalf("changed a root publication: %s", op.store.String())
		}
		name := op.name
		address := op.store.Operand(1)
		if name != "slot" && name != "retained" {
			if address != op.address {
				t.Fatalf("changed nonconstant/nonstack address %s", name)
			}
			continue
		}
		if address == op.address || address.InstructionParent() != op.store.InstructionParent() {
			t.Fatalf("did not localize %s", name)
		}
	}
	body := fn.String()
	if !strings.Contains(body, "load ptr, ptr %retained") ||
		!strings.Contains(body, "store ptr %retained, ptr @escape") ||
		strings.Contains(body, "%unused =") || strings.Contains(body, "%slot =") {
		t.Fatalf("incorrect handling of retained or dead addresses:\n%s", body)
	}
	for _, input := range []string{
		"define void @f() { ret void }",
		"declare i32 @setjmp(ptr) returns_twice\ndefine void @f() { ret void }",
	} {
		mod := parseWasmAggregateIR(t, input)
		before := mod.String()
		if moved := localizeWasmStackAddresses("wasm", mod); moved != 0 || mod.String() != before {
			t.Fatal("changed a module without setjmp calls")
		}
	}
}
