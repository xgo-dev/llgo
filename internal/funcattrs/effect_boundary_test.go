package funcattrs

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestImplicitEffectsCheckOnlyRetainedRestrictions(t *testing.T) {
	for _, tc := range []struct {
		effect     string
		restricted bool
	}{
		{"access(read)", true}, {"access(write)", true}, {"access(readwrite)", false},
		{"capture(none)", true}, {"capture(any)", false}, {"noalias", true},
	} {
		t.Run(tc.effect, func(t *testing.T) {
			mod := valueTestModule(t, "declare void @F(ptr)")
			fn := mod.NamedFunction("F")
			attachValueTestContracts(t, mod, "F", "//llgo:attr param(p) "+tc.effect+"\nfunc F(p *int) {}")
			err := CheckInstrumentation(fn, "runtime operation", "access", "captures", "noalias")
			if (err != nil) != tc.restricted || err != nil && !strings.Contains(err.Error(), "runtime operation") {
				t.Fatalf("restriction check: %v", err)
			}
			WidenForUnknownInstrumentation(fn)
			if err = CheckInstrumentation(fn, "runtime operation", "access", "captures", "noalias"); err != nil {
				t.Fatalf("removed restriction still rejected: %v", err)
			}
		})
	}
}

func TestReturnCaptureUsesMemoryAfterSretAtDefinitionAndCall(t *testing.T) {
	mod := valueTestModule(t, `
declare ptr @logical(ptr)
declare void @physical(ptr, ptr)
define void @caller(ptr %out, ptr %p) {
  %value = call ptr @logical(ptr %p)
  call void @physical(ptr %out, ptr %p)
  ret void
}
`)
	attrs, sig, err := parseTest(t, `//llgo:attr memory(none)
//llgo:attr param(p) capture(results) noalias
//llgo:attr result(0) same_as(param(p))
func F(p *int) *int { return p }`)
	if err != nil {
		t.Fatal(err)
	}
	old, physical := mod.NamedFunction("logical"), mod.NamedFunction("physical")
	if err = Apply(mod.Context(), old, sig, attrs, 0, 64); err != nil {
		t.Fatal(err)
	}
	oldCall := mod.NamedFunction("caller").EntryBasicBlock().FirstInstruction()
	newCall := llvm.NextInstruction(oldCall)
	for _, index := range []int{-1, 1} {
		for _, attr := range old.GetAttributesAtIndex(index) {
			oldCall.AddCallSiteAttribute(index, attr)
		}
	}
	// Model the generic copy made by ABI lowering before the semantic remap.
	for _, attr := range old.GetAttributesAtIndex(1) {
		physical.AddAttributeAtIndex(2, attr)
	}
	newCall.AddCallSiteAttribute(2, old.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("captures")))
	for _, attr := range old.GetAttributesAtIndex(-1) {
		physical.AddFunctionAttr(attr)
	}
	mapping := ABIMapping{Result: ABIValue{Kind: Indirect, Indices: []int{1}}, Params: []ABIValue{DirectValue(2)}}
	if err = RemapFunction(old, physical, mapping); err != nil {
		t.Fatal(err)
	}
	RemapCallEffects(oldCall, newCall, mapping)
	if !physical.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("captures")).IsNil() {
		t.Fatal("definition kept return-only capture through sret")
	}
	if got := newCall.GetCallSiteEnumAttribute(2, llvm.AttributeKindID("captures")).GetEnumValue(); got != nativeAllCapture {
		t.Fatalf("call capture = %#x", got)
	}
	if got := newCall.GetCallSiteEnumAttribute(-1, llvm.AttributeKindID("memory")).GetEnumValue(); got != uint64(AccessWrite) {
		t.Fatalf("call omitted sret write: %#x", got)
	}
	var records []EffectLowering
	if err = json.Unmarshal([]byte(physical.GetStringAttributeAtIndex(-1, EffectsMetadata).GetStringValue()), &records); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range records {
		if record.Name == "capture" {
			found = true
			if record.Disposition != EffectConservative || record.PhysicalIndex != 2 {
				t.Fatalf("capture disposition: %+v", record)
			}
		}
	}
	if !found {
		t.Fatal("missing capture disposition")
	}
	if err = llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	WidenForGCRootPublication(physical)
	if got := physical.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).GetEnumValue(); got != nativeAllMemory&^nativeArgumentMemory|uint64(AccessWrite) {
		t.Fatalf("GC root writes omitted: %#x", got)
	}
}

func TestPointerEffectsDoNotAnnotatePackedStorage(t *testing.T) {
	for _, signature := range []string{"declare void @F(i64)", "declare void @F()"} {
		mod := valueTestModule(t, signature)
		attrs, sig, err := parseTest(t, "//llgo:attr param(p) noalias access(read) capture(none)\nfunc F(p *int) {}")
		if err != nil {
			t.Fatal(err)
		}
		fn := mod.NamedFunction("F")
		if err = ApplyEffects(mod.Context(), fn, sig, attrs, 0, 64); err != nil {
			t.Fatal(err)
		}
		var records []EffectLowering
		if err = json.Unmarshal([]byte(fn.GetStringAttributeAtIndex(-1, EffectsMetadata).GetStringValue()), &records); err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if record.Disposition != EffectConservative {
				t.Fatalf("restriction attached to transport: %+v", record)
			}
		}
		if len(records) != 3 {
			t.Fatalf("missing source effects: %+v", records)
		}
		if err = llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
}
