package funcattrs

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLLVMInfersExecutionAttributesWithoutSourceAnnotations(t *testing.T) {
	mod := valueTestModule(t, `
declare i32 @unknown(i32)
define i32 @add_one(i32 %x) {
  %r = add i32 %x, 1
  ret i32 %r
}
define i32 @calls_unknown(i32 %x) {
  %r = call i32 @unknown(i32 %x)
  ret i32 %r
}
define void @forever() {
  br label %loop
loop:
  br label %loop
}
`)
	names := []string{"nofree", "nosync", "nounwind", "willreturn"}
	for _, name := range names {
		if !mod.NamedFunction("add_one").GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil() {
			t.Fatalf("%s unexpectedly present before inference", name)
		}
	}
	optimizeValueTest(t, mod)
	for _, name := range names {
		kind := llvm.AttributeKindID(name)
		if mod.NamedFunction("add_one").GetEnumFunctionAttribute(kind).IsNil() {
			t.Errorf("LLVM did not infer %s from the function body", name)
		}
		if !mod.NamedFunction("calls_unknown").GetEnumFunctionAttribute(kind).IsNil() {
			t.Errorf("LLVM assumed %s across an unknown external call", name)
		}
	}
	if !mod.NamedFunction("forever").GetEnumFunctionAttribute(llvm.AttributeKindID("willreturn")).IsNil() {
		t.Fatal("an infinite loop was marked willreturn")
	}
}

func TestABIEffectsComposeTransportAndCapture(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("abi-effects")
	defer mod.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	aggregate := llvm.ArrayType(ctx.Int64Type(), 8)
	old := llvm.AddFunction(mod, "logical", llvm.FunctionType(aggregate, []llvm.Type{aggregate, ptr, ptr}, false))
	old.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("memory"), 0))
	old.AddAttributeAtIndex(2, ctx.CreateEnumAttribute(llvm.AttributeKindID("captures"), 0xf))
	old.AddAttributeAtIndex(3, ctx.CreateEnumAttribute(llvm.AttributeKindID("captures"), 0))
	old.AddAttributeAtIndex(3, ctx.CreateEnumAttribute(llvm.AttributeKindID("readonly"), 0))
	physical := llvm.AddFunction(mod, "physical", llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr, ptr, ptr}, false))
	// Simulate generic native-attribute copying before the dedicated mapper.
	physical.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("memory"), 0))
	physical.AddAttributeAtIndex(3, ctx.CreateEnumAttribute(llvm.AttributeKindID("captures"), 0xf))
	mapping := ABIMapping{
		Result: ABIValue{Kind: Indirect, Indices: []int{1}},
		Params: []ABIValue{{Kind: Indirect, Indices: []int{2}}, DirectValue(3), DirectValue(4)},
	}
	RemapFunctionEffects(old, physical, mapping)
	if got := physical.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).GetEnumValue(); got != 3 {
		t.Fatalf("transport effect = %#x, want byval read plus sret write", got)
	}
	if !physical.GetEnumAttributeAtIndex(3, llvm.AttributeKindID("captures")).IsNil() {
		t.Fatal("return-only capture incorrectly survived an sret store")
	}
	if physical.GetEnumAttributeAtIndex(4, llvm.AttributeKindID("captures")).IsNil() || physical.GetEnumAttributeAtIndex(4, llvm.AttributeKindID("readonly")).IsNil() {
		t.Fatal("unrelated direct pointer effects were lost")
	}
	if !physical.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("readonly")).IsNil() {
		t.Fatal("logical pointer permissions were attached to a carrier")
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestABIHiddenPointerWidensMemoryLocations(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("hidden-scalar")
	defer mod.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	old := llvm.AddFunction(mod, "logical", llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false))
	old.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("memory"), 1))
	old.AddAttributeAtIndex(1, ctx.CreateEnumAttribute(llvm.AttributeKindID("readonly"), 0))
	physical := llvm.AddFunction(mod, "physical", old.GlobalValueType())
	RemapFunctionEffects(old, physical, ABIMapping{Result: DirectValue(0), Params: []ABIValue{{Kind: Indirect, Indices: []int{1}}}})
	if got := physical.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).GetEnumValue(); got != 0x555 {
		t.Fatalf("hidden pointer effect = %#x, want conservative memory(read)", got)
	}
	if !physical.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("readonly")).IsNil() {
		t.Fatal("pointee readonly leaked onto its transport slot")
	}
}

func TestUnknownInstrumentationKeepsValuesAndNoNormalReturn(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("instrumentation")
	defer mod.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	fn := llvm.AddFunction(mod, "instrumented", llvm.FunctionType(ptr, []llvm.Type{ptr}, false))
	for _, name := range []string{"memory", "nofree", "nosync", "nounwind", "willreturn", "cold", "noreturn"} {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID(name), 0))
	}
	fn.AddAttributeAtIndex(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0))
	fn.AddAttributeAtIndex(1, ctx.CreateEnumAttribute(llvm.AttributeKindID("captures"), 0))
	fn.AddAttributeAtIndex(1, ctx.CreateEnumAttribute(llvm.AttributeKindID("readonly"), 0))
	WidenForUnknownInstrumentation(fn)
	for _, name := range []string{"memory", "nofree", "nosync", "nounwind", "willreturn"} {
		if !fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name)).IsNil() {
			t.Errorf("unsafe %s retained", name)
		}
	}
	for _, name := range []string{"cold", "noreturn"} {
		if fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name)).IsNil() {
			t.Errorf("safe %s lost", name)
		}
	}
	if fn.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil() || strings.Contains(fn.String(), "captures") || strings.Contains(fn.String(), "readonly") {
		t.Fatalf("incorrect instrumented declaration:\n%s", fn.String())
	}
}
