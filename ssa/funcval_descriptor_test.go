package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/runtime/abi"
	ssaabi "github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

func TestFuncvalDescriptorLayout(t *testing.T) {
	for _, target := range []Target{
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			if target.GOARCH == "wasm" {
				prog.TypeSizes(&types.StdSizes{WordSize: 8, MaxAlign: 8})
			}
			fn := prog.Closure(NoArgsNoRet)
			word := uint64(prog.GoWordSize())
			if fn.kind != vkClosure || fn.ll.TypeKind() != llvm.PointerTypeKind || prog.SizeOf(fn) != word {
				t.Fatalf("Go function value is not one descriptor pointer: kind=%d LLVM=%s size=%d", fn.kind, fn.ll, prog.SizeOf(fn))
			}
			if got := prog.td.TypeAllocSize(prog.storageType(fn)); got != word {
				t.Fatalf("LLVM function storage is %d bytes, want %d", got, word)
			}
			box := prog.Struct(fn, prog.Byte())
			if got := prog.OffsetOf(box, 1); got != word {
				t.Fatalf("field after function starts at %d, want %d", got, word)
			}
			if got := prog.SizeOf(prog.funcvalDescriptorType(fn)); got != 2*word {
				t.Fatalf("descriptor header is %d bytes, want %d", got, 2*word)
			}
			if prog.abi.Kind(fn.raw.Type) != abi.Func || prog.abi.TFlag(fn.raw.Type)&abi.TFlagClosure == 0 || prog.abi.PtrBytes(fn.raw.Type) != uintptr(word) {
				t.Fatal("function metadata disagrees with one-word descriptor storage")
			}
			kind, _, level := ssaabi.DataKindOf(fn.raw.Type, 0, prog.is32Bits)
			if kind != ssaabi.Pointer || level != 0 || !directIfaceType(fn.raw.Type) {
				t.Fatal("interface payload does not carry the descriptor pointer directly")
			}
			native := prog.Type(NoArgsNoRet, CDecl)
			if native.kind != vkFuncPtr || prog.abi.TFlag(native.raw.Type)&abi.TFlagClosure != 0 {
				t.Fatal("native entry pointer acquired the Go descriptor role")
			}
		})
	}
}

func TestFuncvalDescriptorConstantOwners(t *testing.T) {
	for _, target := range []Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
	} {
		for _, mode := range []string{"shared", "private", "private-gep"} {
			t.Run(target.GOOS+"/"+mode, func(t *testing.T) {
				prog := NewProgram(&target)
				defer prog.Dispose()
				prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
				if target.GOARCH == "wasm" {
					prog.TypeSizes(&types.StdSizes{WordSize: 8, MaxAlign: 8})
				}
				var modules [2]llvm.Module
				for i, path := range []string{"example.com/a", "example.com/b"} {
					pkg := prog.NewPackage(path, path)
					modules[i] = pkg.Module()
					entry := pkg.NewFunc("shared.entry", NoArgsNoRet, GoDecl)
					array := llvm.ArrayType(prog.tyInt8(), 1)
					ctx := llvm.AddGlobal(pkg.Module(), array, "ctx")
					ctx.SetGlobalConstant(true)
					if mode != "shared" {
						ctx.SetLinkage(llvm.PrivateLinkage)
						ctx.SetInitializer(llvm.ConstArray(prog.tyInt8(), []llvm.Value{llvm.ConstInt(prog.tyInt8(), uint64(i+1), false)}))
					}
					env := ctx
					if mode == "private-gep" {
						zero := llvm.ConstInt(prog.tyInt32(), 0, false)
						env = llvm.ConstGEP(array, ctx, []llvm.Value{zero, zero})
					}
					result := types.NewTuple(types.NewVar(token.NoPos, nil, "", NoArgsNoRet))
					sig := types.NewSignatureType(nil, nil, nil, nil, result, false)
					caller := pkg.NewFunc(path+".export", sig, GoDecl)
					body := caller.MakeBody(1)
					value := body.makeFuncval(prog.Closure(NoArgsNoRet), entry.impl, env)
					body.Return(value)
				}
				if err := llvm.LinkModules(modules[0], modules[1]); err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(modules[0], llvm.ReturnStatusAction); err != nil {
					t.Fatalf("linked descriptor modules are invalid: %v", err)
				}
				count := 0
				for global := modules[0].FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
					if strings.HasPrefix(global.Name(), "_llgo_funcval$") {
						count++
					}
				}
				want := 2
				if mode == "shared" {
					want = 1
				}
				if count != want {
					t.Fatalf("linked descriptors=%d, want %d; private captures must not be ODR merged", count, want)
				}
			})
		}
	}
}

// Observe the actual header producer rather than interpreting a descriptor
// pointer as an LLVM aggregate. Dynamic pointers must be stored in the right
// field of collectible header storage; constant operands live in its initializer.
func assertFuncvalHeaderPart(t *testing.T, owner Function, value Expr, index int, want llvm.Value) {
	t.Helper()
	if global := value.impl.IsAGlobalVariable(); !global.IsNil() {
		if got := global.Initializer().Operand(index); got != want {
			t.Fatalf("descriptor field %d = %s, want %s", index, got, want)
		}
		return
	}
	header := owner.Prog.funcvalDescriptorType(value.Type).ll
	for block := owner.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			store := instruction.IsAStoreInst()
			if store.IsNil() || store.Operand(0) != want {
				continue
			}
			ptr := store.Operand(1).IsAGetElementPtrInst()
			if ptr.IsNil() || ptr.GEPSourceElementType() != header || ptr.OperandsCount() != 3 {
				continue
			}
			field := ptr.Operand(2).IsAConstantInt()
			if !field.IsNil() && field.ZExtValue() == uint64(index) {
				return
			}
		}
	}
	t.Fatalf("descriptor field %d did not retain %s:\n%s", index, want, owner.impl)
}

func assertFuncvalCollectibleAllocations(t *testing.T, owner Function, want int) {
	t.Helper()
	count := 0
	for block := owner.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			call := instruction.IsACallInst()
			if call.IsNil() {
				continue
			}
			entry := call.CalledValue().IsAFunction()
			if entry.IsNil() || !strings.HasPrefix(entry.Name(), PkgRuntime+".Alloc") {
				continue
			}
			if entry.Name() != PkgRuntime+".AllocU" && entry.Name() != PkgRuntime+".AllocUAligned" && entry.Name() != PkgRuntime+".AllocAtomicU" && entry.Name() != PkgRuntime+".AllocAtomicUAligned" {
				t.Fatalf("descriptor uses an unexpected allocator %s", entry.Name())
			}
			count++
		}
	}
	if count != want {
		t.Fatalf("descriptor has %d collectible allocation sites, want %d:\n%s", count, want, owner.impl)
	}
}
