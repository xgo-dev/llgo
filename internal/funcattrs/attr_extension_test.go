package funcattrs

import (
	"strings"
	"testing"
)

func TestAttrSyntaxAndNoAlias(t *testing.T) {
	attrs, sig, err := parseTest(t, `// llgo:attr cold
//llgo:attr param(p) noalias
func F(p *int) { *p = 1 }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 2 {
		t.Fatalf("new attributes were ignored: %v", attrs)
	}
	if err := Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
}

func TestAttrNoAliasValidation(t *testing.T) {
	for _, tc := range []struct{ directive, sig string }{
		{"result(0) noalias", "func F() *int { return nil }"},
		{"noalias", "func F() {}"},
		{"param(p) noalias", "func F(p int) {}"},
	} {
		attrs, sig, err := parseTest(t, "//llgo:attr "+tc.directive+"\n"+tc.sig)
		if err == nil {
			err = Validate(attrs, sig, 64, false)
		}
		if err == nil {
			t.Errorf("accepted %s", tc.directive)
		}
	}
}

func TestAttrRejectsOldSpelling(t *testing.T) {
	_, _, err := parseTest(t, "//llgo:attribute cold\nfunc F() {}")
	if err == nil || !strings.Contains(err.Error(), "use //llgo:attr") {
		t.Fatalf("old spelling diagnostic = %v", err)
	}
}
