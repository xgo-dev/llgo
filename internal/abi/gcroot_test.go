package abi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLargeAggregateTemporaryRoots(t *testing.T) {
	const input = `
%Large = type [65537 x i8]
declare void @collect()
define %Large @make(ptr %src) {
entry:
  call void @collect()
  %value = load %Large, ptr %src
  ret %Large %value
}
define void @snapshot(ptr %src, ptr %dst) {
entry:
  %value = load %Large, ptr %src
  call void @collect()
  store %Large %value, ptr %dst
  ret void
}
define void @caller(ptr %src, ptr %dst) {
entry:
  %value = call %Large @make(ptr %src)
  call void @collect()
  store %Large %value, ptr %dst
  ret void
}
`
	for _, tc := range []struct {
		name, layout string
		goWordSize   int
	}{
		{"wasm32-go64", "e-p:32:32-i64:64-n32:64-S128", 8},
		{"wasm64-go64", "e-p:64:64-i64:64-n32:64-S128", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "roots.ll")
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
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
			defer mod.Dispose()
			td := llvm.NewTargetData(tc.layout)
			defer td.Dispose()
			// Exercise nesting with a frame already emitted by the frontend.
			fn := mod.NamedFunction("snapshot")
			frame := NewGCRootFrame(mod, fn, 2, td.PointerSize(), tc.goWordSize, true)
			PopGCRootFrame(mod, fn, frame)
			LowerLargeAggregates(td, mod, AggregateLoweringConfig{GoWordSize: tc.goWordSize, GCRoots: true, Wasm: true})
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid rooted lowering: %v\n%s", err, mod.String())
			}
			for _, name := range []string{"make", "snapshot", "caller"} {
				fn := mod.NamedFunction(name)
				ir := fn.String()
				if !strings.Contains(ir, "@llvm_gc_root_sjlj_replaying") || !strings.Contains(ir, "select i1") {
					t.Fatalf("%s lacks replay-safe root push/pop:\n%s", name, ir)
				}
				if name == "make" {
					if !hasPointerRootStore(fn.Param(0)) {
						t.Fatal("sret storage is not rooted by callee")
					}
					continue
				}
				found := false
				for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
					for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
						if !instr.IsACallInst().IsNil() && instr.CalledValue().Name() == runtimeAllocU {
							found = true
							if !hasPointerRootStore(instr) {
								t.Fatalf("%s allocation is not rooted", name)
							}
						}
					}
				}
				if !found {
					t.Fatalf("%s did not exercise a lowering-created allocation", name)
				}
			}
			allocType := mod.NamedFunction(runtimeAllocU).GlobalValueType()
			if got := allocType.ParamTypes()[0].IntTypeWidth(); got != tc.goWordSize*8 {
				t.Fatalf("AllocU parameter width = %d, want %d", got, tc.goWordSize*8)
			}
			rootStorage := mod.NamedGlobal("llvm_gc_root_chain").GlobalValueType()
			if got := rootStorage.TypeKind() == llvm.StructTypeKind; got != (tc.goWordSize > td.PointerSize()) {
				t.Fatalf("wide root storage = %t, want %t", got, tc.goWordSize > td.PointerSize())
			}
			// The native frame-map shape remains supported by the shared emitter.
			fn = mod.NamedFunction("caller")
			frame = NewGCRootFrame(mod, fn, 1, td.PointerSize(), tc.goWordSize, false)
			PopGCRootFrame(mod, fn, frame)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func hasPointerRootStore(value llvm.Value) bool {
	seen := make(map[llvm.Value]bool)
	var walk func(llvm.Value) bool
	walk = func(value llvm.Value) bool {
		if seen[value] {
			return false
		}
		seen[value] = true
		for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
			user := use.User()
			if store := user.IsAStoreInst(); !store.IsNil() && store.Operand(0) == value {
				return true
			}
			if !user.IsAInsertValueInst().IsNil() && walk(user) {
				return true
			}
		}
		return false
	}
	return walk(value)
}
