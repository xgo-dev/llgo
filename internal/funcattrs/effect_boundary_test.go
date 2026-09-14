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
		{"noalias", true},
	} {
		t.Run(tc.effect, func(t *testing.T) {
			mod := valueTestModule(t, "declare void @F(ptr)")
			fn := mod.NamedFunction("F")
			attachValueTestContracts(t, mod, "F", "//llgo:param(p) "+tc.effect+"\nfunc F(p *int) {}")
			err := CheckInstrumentation(fn, "runtime operation", "access", "noalias")
			if (err != nil) != tc.restricted || err != nil && !strings.Contains(err.Error(), "runtime operation") {
				t.Fatalf("restriction check: %v", err)
			}
			WidenForUnknownInstrumentation(fn)
			if err = CheckInstrumentation(fn, "runtime operation", "access", "noalias"); err != nil {
				t.Fatalf("removed restriction still rejected: %v", err)
			}
		})
	}
}

func TestPointerEffectsDoNotAnnotatePackedStorage(t *testing.T) {
	for _, signature := range []string{"declare void @F(i64)", "declare void @F()"} {
		mod := valueTestModule(t, signature)
		attrs, sig, err := parseTest(t, "//llgo:param(p) noalias access(read)\nfunc F(p *int) {}")
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
		if len(records) != 2 {
			t.Fatalf("missing source effects: %+v", records)
		}
		if err = llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
}
