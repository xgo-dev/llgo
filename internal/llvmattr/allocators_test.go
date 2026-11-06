package llvmattr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestAllocatorContracts(t *testing.T) {
	for _, bits := range []int{1, 32, 64, 65, 128} {
		for _, roots := range []bool{false, true} {
			t.Run(fmt.Sprintf("i%d/roots=%v", bits, roots), func(t *testing.T) {
				ctx := llvm.NewContext()
				defer ctx.Dispose()
				mod := ctx.NewModule("allocators")
				defer mod.Dispose()
				b := ctx.NewBuilder()
				defer b.Dispose()
				intType := ctx.IntType(bits)
				ptrType := llvm.PointerType(ctx.Int8Type(), 0)
				fnType := llvm.FunctionType(ptrType, []llvm.Type{intType}, false)
				for _, name := range []string{"AllocU", "AllocZ", "AllocNoScanU", "AllocNoScanZ", "AllocRoot", "AllocUAligned", "unknown"} {
					fn := llvm.AddFunction(mod, runtimePrefix+name, fnType)
					ApplyAllocatorFunction(ctx, fn)
					wantNonNull := name == "AllocU" || name == "AllocZ" || name == "AllocNoScanU" || name == "AllocNoScanZ" || name == "AllocRoot"
					if got := !fn.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil(); got != wantNonNull {
						t.Fatalf("%s: nonnull = %v", name, got)
					}
					wantSize := name == "AllocU" || name == "AllocZ"
					if got := strings.Contains(fn.String(), "allocsize(0)"); got != wantSize {
						t.Fatalf("wrong size contract: %s", fn.String())
					}
					if !fn.GetEnumFunctionAttribute(llvm.AttributeKindID("allockind")).IsNil() ||
						!fn.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("noalias")).IsNil() {
						t.Fatalf("zero-size declaration promises fresh storage: %s", fn.String())
					}
					caller := llvm.AddFunction(mod, "call"+name, fnType)
					b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
					for i, size := range []llvm.Value{
						llvm.ConstInt(intType, 0, false), caller.Param(0),
						llvm.ConstInt(intType, 16, false),
						llvm.ConstInt(intType, uint64(1)<<(bits-1), false),
					} {
						args := []llvm.Value{size}
						call := b.CreateCall(fnType, fn, args, "")
						ApplyAllocatorCall(ctx, call, fn, args, roots)
						attr := call.GetCallSiteEnumAttribute(-1, llvm.AttributeKindID("allockind"))
						wantKind := wantSize && !roots && i == 2 && bits >= 2 && bits <= 64
						if got := !attr.IsNil(); got != wantKind {
							t.Fatalf("%s size %s: allockind = %v", name, size.String(), got)
						}
						if wantKind {
							want := `allockind("alloc,uninitialized")`
							if name == "AllocZ" {
								want = `allockind("alloc,zeroed")`
							}
							// Check LLVM's spelling, not a duplicate of our bit encoding.
							if !strings.Contains(mod.String(), want) {
								t.Fatalf("missing %s:\n%s", want, mod.String())
							}
						}
					}
					b.CreateRet(llvm.ConstNull(ptrType))
				}
				// Neither a same-named user function nor an indirect callee inherits
				// the trusted runtime symbol's contract.
				user := llvm.AddFunction(mod, "user.AllocZ", fnType)
				ApplyAllocatorFunction(ctx, user)
				caller := llvm.AddFunction(mod, "indirect", llvm.FunctionType(ptrType, []llvm.Type{ptrType}, false))
				b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
				for _, callee := range []llvm.Value{user, caller.Param(0)} {
					args := []llvm.Value{llvm.ConstInt(intType, 16, false)}
					call := b.CreateCall(fnType, callee, args, "")
					ApplyAllocatorCall(ctx, call, callee, args, false)
					if !call.GetCallSiteEnumAttribute(-1, llvm.AttributeKindID("allockind")).IsNil() {
						t.Fatal("untrusted callee received allocator contract")
					}
				}
				b.CreateRet(llvm.ConstNull(ptrType))
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
