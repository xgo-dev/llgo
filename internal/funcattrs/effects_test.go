package funcattrs

import (
	"encoding/json"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestMemoryEffectGrammar(t *testing.T) {
	for _, test := range []struct {
		text string
		want MemoryEffects
	}{
		{"none", MemoryEffects{}},
		{"read", MemoryEffects{AccessRead, AccessRead}},
		{"args: write", MemoryEffects{AccessWrite, AccessNone}},
		{"read, args: readwrite", MemoryEffects{AccessReadWrite, AccessRead}},
		{"other: write, args: read", MemoryEffects{AccessRead, AccessWrite}},
		{"args: none, readwrite", MemoryEffects{AccessNone, AccessReadWrite}},
	} {
		t.Run(test.text, func(t *testing.T) {
			got, err := ParseMemoryEffects(test.text)
			if err != nil || got != test.want {
				t.Fatalf("effects = %+v, %v; want %+v", got, err, test.want)
			}
		})
	}
	for _, text := range []string{"", "read,write", "args:read,args:read", "other:none,other:write", "global:read", "read,", "args:unknown"} {
		if _, err := ParseMemoryEffects(text); err == nil {
			t.Errorf("accepted malformed effects %q", text)
		}
	}
}

func TestSourceEffectsKeepAggregatePointerSubject(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("effects")
	defer mod.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	aggregate := ctx.StructType([]llvm.Type{ptr, ctx.Int64Type()}, false)
	fn := llvm.AddFunction(mod, "hidden", llvm.FunctionType(ctx.VoidType(), []llvm.Type{aggregate}, false))
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "arg", types.NewStruct([]*types.Var{
		types.NewField(0, nil, "P", types.Typ[types.UnsafePointer], false),
		types.NewField(0, nil, "N", types.Typ[types.Int64], false),
	}, nil))), nil, false)
	attrs := []Attribute{{Target: Target{Scope: Function}, Name: "memory", Memory: MemoryEffects{Args: AccessRead}}}
	if err := ApplyEffects(ctx, fn, sig, attrs, 0, 64); err != nil {
		t.Fatal(err)
	}
	if got := fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).GetEnumValue(); got != 0x555 {
		t.Fatalf("hidden pointer access = %#x; want reads beyond native argmem", got)
	}
	var lowering []EffectLowering
	metadata := fn.GetStringAttributeAtIndex(-1, EffectsMetadata)
	if err := json.Unmarshal([]byte(metadata.GetStringValue()), &lowering); err != nil || len(lowering) != 1 || lowering[0].Disposition != EffectConservative {
		t.Fatalf("lowering = %+v, %v", lowering, err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
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

func TestAssumptionsAndExternalObservationsHaveDifferentEffects(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("observable-effects")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	ft := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int32])),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int32])), false)
	pure := llvm.AddFunction(mod, "pure", ft)
	for _, name := range []string{"noinline", "nounwind", "willreturn"} {
		pure.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID(name), 0))
	}
	if err := ApplyEffects(ctx, pure, sig, []Attribute{{Target: Target{Scope: Function}, Name: "memory", Memory: MemoryEffects{}}}, 0, 64); err != nil {
		t.Fatal(err)
	}
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(pure, "entry"))
	positive := b.CreateICmp(llvm.IntSGT, pure.Param(0), llvm.ConstNull(i32), "")
	b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.assume"), []llvm.Value{positive}, "")
	b.CreateRet(b.CreateMul(pure.Param(0), pure.Param(0), ""))

	// A changing external observation is an event in the source state model:
	// it requires other:readwrite even if it does not modify program storage.
	external := llvm.AddFunction(mod, "observe_external", ft)
	if err := ApplyEffects(ctx, external, sig, []Attribute{{Target: Target{Scope: Function}, Name: "memory", Memory: MemoryEffects{Other: AccessReadWrite}}}, 0, 64); err != nil {
		t.Fatal(err)
	}
	for _, callee := range []llvm.Value{pure, external} {
		caller := llvm.AddFunction(mod, "use_"+callee.Name(), ft)
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
		x := b.CreateCall(ft, callee, []llvm.Value{caller.Param(0)}, "")
		y := b.CreateCall(ft, callee, []llvm.Value{caller.Param(0)}, "")
		b.CreateRet(b.CreateSub(x, y, ""))
	}
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	options.SetVerifyEach(true)
	if err := mod.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
		t.Fatal(err)
	}
	if ir := mod.NamedFunction("use_pure").String(); !strings.Contains(ir, "ret i32 0") {
		t.Fatalf("a source assumption prevented pure-call elimination:\n%s", ir)
	}
	if ir := mod.NamedFunction("pure").String(); !strings.Contains(ir, "@llvm.assume") || mod.NamedFunction("pure").GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).GetEnumValue() != 0 {
		t.Fatalf("contract assumption changed observable storage effects:\n%s", ir)
	}
	if ir := mod.NamedFunction("use_observe_external").String(); strings.Count(ir, "@observe_external") != 2 {
		t.Fatalf("changing external observations were incorrectly merged:\n%s", ir)
	}
}
