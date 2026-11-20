package cabi_test

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/heap2stack"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWasmStringBoundaryPadding(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Goos, conf.Goarch, conf.Tags = "wasip1", "wasm", "nogc"
	pkgs, err := build.Do([]string{"./_testdata/wasmstring/strings.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	ctx := mod.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()

	// Simulate C after ABI lowering, so an outgoing Go conversion cannot
	// accidentally hide a missing incoming conversion. C may leave these
	// padding bytes dirty; the low pointer words and lengths stay intact.
	checks := []string{"LocalStringStorage", "ImportedGlobal"}
	global := mod.NamedGlobal("foreignGlobal")
	if global.IsNil() {
		t.Fatal("missing C GoString global")
	}
	global.SetGlobalConstant(true)
	layout := global.GlobalValueType()
	slot := layout.StructElementTypes()[0]
	parts := slot.StructElementTypes()
	global.SetInitializer(llvm.ConstNamedStruct(layout, []llvm.Value{
		llvm.ConstNamedStruct(slot, []llvm.Value{
			llvm.ConstIntToPtr(llvm.ConstInt(ctx.Int32Type(), 0x6000, false), parts[0]),
			llvm.ConstInt(parts[1], 0xa5a5a5a5, false),
		}),
		llvm.ConstInt(layout.StructElementTypes()[1], 19, false),
	}))
	// Exported entries install a runtime-local context. Its behavior is
	// independent of string representation; provide inert bodies so LLVM can
	// evaluate the value checks without linking or executing a Wasm runtime.
	for _, name := range []string{"EnterLocalContext", "LeaveLocalContext"} {
		fn := mod.NamedFunction(ssa.PkgRuntime + "." + name)
		if !fn.IsNil() && fn.IsDeclaration() {
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "stub"))
			if fn.GlobalValueType().ReturnType().TypeKind() == llvm.VoidTypeKind {
				b.CreateRetVoid()
			} else {
				b.CreateRet(llvm.ConstNull(fn.GlobalValueType().ReturnType()))
			}
		}
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !fn.IsDeclaration() {
			fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("alwaysinline"), 0))
		}
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	pbo := llvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	pbo.SetVerifyEach(true)
	// Inline the observers and fold unused header projections before escape
	// analysis, so it can promote the address-taken aggregates in the fixture.
	if err := mod.RunPasses("always-inline,function(instcombine)", pkg.Prog.TargetMachine(), pbo); err != nil {
		t.Fatal(err)
	}
	if err := heap2stack.Run(mod, pkg.Prog.TargetMachine()); err != nil {
		t.Fatal(err)
	}
	if err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), pbo); err != nil {
		t.Fatal(err)
	}
	// Check behavior rather than a particular insertvalue/mask sequence.
	// Both native directions must preserve every data field while clearing
	// only string and slice padding.
	for _, name := range checks {
		fn := mod.NamedFunction(name)
		if fn.IsNil() {
			t.Errorf("%s disappeared after optimization", name)
			continue
		}
		returns := 0
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			if ret := bb.LastInstruction(); !ret.IsNil() && !ret.IsAReturnInst().IsNil() {
				returns++
				if value := ret.Operand(0); value.IsAConstantInt().IsNil() || value.ZExtValue() != 1 {
					t.Errorf("%s did not preserve the expected string value:\n%s", name, fn.String())
				}
			}
		}
		if returns == 0 {
			t.Errorf("%s has no return after optimization:\n%s", name, fn.String())
		}
	}
}
