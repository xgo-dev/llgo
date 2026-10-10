package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestRuntimeAttributes(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	rt, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, roots := range []bool{false, true} {
		prog := NewProgram(nil)
		prog.SetRuntime(rt)
		defer prog.Dispose()
		prog.EnableGCRoots(roots)
		for _, module := range []string{PkgRuntime, "example.com/caller"} {
			pkg := prog.NewPackage("p", module)
			for _, tc := range []struct {
				name  string
				index int
				attr  string
				want  bool
			}{
				{"AllocU", 0, "nonnull", true}, {"AllocZ", -1, "allocsize", true},
				{"AllocRoot", 0, "noalias", false}, {"CStrDup", 0, "nonnull", true},
				{"StringEqual", -1, "memory", !roots}, {"StringLess", -1, "willreturn", !roots},
				{"CStrCopy", 1, "returned", !roots}, {"StringFrom", 1, "readonly", !roots},
				{"Panic", -1, "noreturn", true}, {"PanicIndex", -1, "noreturn", true},
				{"PanicSIMDUnimplemented", -1, "noreturn", true},
				{"ChanCap", -1, "memory", !roots}, {"MapLen", -1, "memory", !roots},
				{"ChanLen", -1, "memory", false}, {"PanicWrapNilPointer", -1, "noreturn", false},
				{"AssertNilDeref", -1, "noreturn", false}, {"Rethrow", -1, "noreturn", false},
				{"EfaceEqual", -1, "memory", false},
				{"CStrDup", 0, "noalias", true}, {"NewStringIter", 0, "noalias", true},
				{"NewMapIter", 0, "noalias", true}, {"NewChan", 0, "noalias", true},
				{"NewChan", 0, "nonnull", true}, {"MakeMap", 0, "noalias", true},
				{"MakeSmallMap", 0, "nonnull", true}, {"MakeSmallMap", 0, "noalias", true},
				{"AssertNilDerefPtr", 0, "nonnull", true}, {"AssertNilDerefPtr", 1, "returned", true},
				{"AssertNilDerefPtr", 1, "nonnull", false},
				{"New", 0, "noalias", false}, {"MapAccess1", 0, "noalias", false},
				{"MapAssign", 0, "noalias", false}, {"NewItab", 0, "noalias", false},
				{"NewItab", 0, "nonnull", false}, {"CString", 0, "nonnull", false},
			} {
				obj := rt.Scope().Lookup(tc.name)
				if obj == nil {
					t.Fatalf("missing runtime function %s", tc.name)
				}
				fn := pkg.NewFunc(PkgRuntime+"."+tc.name, obj.Type().(*types.Signature), InGo)
				got := !fn.impl.GetEnumAttributeAtIndex(tc.index, llvm.AttributeKindID(tc.attr)).IsNil()
				if got != tc.want {
					t.Errorf("roots=%v %s: %s index %d = %v, want %v", roots, tc.name, tc.attr, tc.index, got, tc.want)
				}
			}
			// An identically named user helper must not inherit a runtime contract.
			fn := pkg.NewFunc("example.com/runtime.AllocZ", prog.tyMalloc(), InGo)
			if !fn.impl.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil() {
				t.Fatal("runtime contract leaked to user function")
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestRuntimeAllocationCallAttributes(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasi"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.EnableGCRoots(true) // Freshness holds even with root instrumentation.
			pkg := prog.NewPackage("p", "example.com/caller")
			for _, name := range []string{"AllocU", "AllocZ", "AllocRoot", "Unknown"} {
				callee := pkg.NewFunc(PkgRuntime+"."+name, prog.tyMalloc(), InGo)
				fn := pkg.NewFunc("check"+name, prog.tyMalloc(), InGo)
				b := fn.MakeBody(1)
				highBit := uint64(1) << (prog.Uintptr().ll.IntTypeWidth() - 1)
				for _, size := range []Expr{prog.IntVal(0, prog.Uintptr()), prog.IntVal(1, prog.Uintptr()),
					prog.IntVal(64, prog.Uintptr()), prog.IntVal(highBit, prog.Uintptr()), fn.Param(0)} {
					call := b.Call(callee.Expr, size)
					constant := size.impl.IsAConstantInt()
					want := name != "Unknown" && !constant.IsNil() && constant.ZExtValue() > 0 && constant.ZExtValue() < highBit
					for _, attrName := range []string{"noalias", "dereferenceable"} {
						attr := call.impl.GetCallSiteEnumAttribute(0, llvm.AttributeKindID(attrName))
						if attr.IsNil() == want {
							t.Fatalf("%s(%s): unexpected %s: %s", name, size.impl.String(), attrName, call.impl.String())
						}
						if attrName == "dereferenceable" && want && attr.GetEnumValue() != constant.ZExtValue() {
							t.Fatal("wrong extent")
						}
					}
				}
				// The declaration must still be valid for zero-size and dynamic requests.
				if !callee.impl.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("noalias")).IsNil() {
					t.Fatal("conditional noalias leaked to declaration")
				}
				b.Return(prog.Nil(prog.VoidPtr()))
				b.EndBuild()
			}
			// Indirect calls have no proven runtime identity, even if their signature
			// matches an allocator and their argument is a positive constant.
			sig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "f", types.Typ[types.UnsafePointer])), nil, false)
			indirect := pkg.NewFunc("indirect", sig, InGo)
			b := indirect.MakeBody(1)
			ty := prog.FuncDecl(prog.tyMalloc(), InGo)
			ptr := Expr{indirect.Param(0).impl, ty}
			call := b.Call(ptr, prog.IntVal(64, prog.Uintptr()))
			if !call.impl.GetCallSiteEnumAttribute(0, llvm.AttributeKindID("noalias")).IsNil() {
				t.Fatal("indirect call gained runtime contract")
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRuntimeAllocationNoAliasOptimization(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/caller")
	callee := pkg.NewFunc(PkgRuntime+".AllocU", prog.tyMalloc(), InGo)
	sig := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.Byte])), false)
	fn := pkg.NewFunc("independent", sig, InGo)
	b := fn.MakeBody(1)
	ptr := prog.Pointer(prog.Byte())
	p := b.Convert(ptr, b.Call(callee.Expr, prog.IntVal(1, prog.Uintptr())))
	q := b.Convert(ptr, b.Call(callee.Expr, prog.IntVal(1, prog.Uintptr())))
	b.Store(p, prog.IntVal(7, prog.Byte()))
	b.Store(q, prog.IntVal(9, prog.Byte()))
	b.Return(b.Load(p))
	b.EndBuild()
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	opts := llvm.NewPassBuilderOptions()
	defer opts.Dispose()
	if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mod.NamedFunction("independent").String(), "ret i8 7") {
		t.Fatalf("fresh allocations still alias:\n%s", mod.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
