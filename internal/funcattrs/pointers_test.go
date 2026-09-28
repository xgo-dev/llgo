package funcattrs

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestPointerAttributeValidation(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"//llgo:param(p) access(read) noalias\nfunc F(p *int) {}", ""},
		{"type T int\n//llgo:receiver access(none)\nfunc (*T) F() {}", ""},
		{"//llgo:param(p) access(read) access(read)\nfunc F(p *int) {}", ""},
		{"//llgo:param(p) access(read) access(write)\nfunc F(p *int) {}", "conflicting"},
		{"//llgo:param(p) access(invalid)\nfunc F(p *int) {}", "unknown access mode"},
		{"//llgo:param(p) noalias(x)\nfunc F(p *int) {}", "invalid arguments"},
		{"//llgo:result noalias\nfunc F() *int { return nil }", "not supported on result"},
		{"//llgo:result access(read)\nfunc F() *int { return nil }", "not supported on result"},
		{"//llgo:param(p) access(read)\nfunc F(p int) {}", "requires a pointer"},
		{"//llgo:param(p) noalias\nfunc F(p []int) {}", "requires a pointer"},
	} {
		attrs, sig, err := parseTest(t, test.source)
		if err == nil {
			err = Validate(attrs, sig, 64, false)
		}
		if test.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: %v, want %s", test.source, err, test.want)
		}
	}
}

func TestPointerAccessDoesNotRestrictOtherMemory(t *testing.T) {
	mod := valueTestModule(t, `
@global = global i32 0
define i32 @read(ptr %p) {
 store i32 7, ptr @global
 %v = load i32, ptr %p
 ret i32 %v
}
define i32 @caller() {
 store i32 1, ptr @global
 %v = call i32 @read(ptr @global)
 %after = load i32, ptr @global
 %sum = add i32 %v, %after
 ret i32 %sum
}
`)
	attrs, sig, err := parseTest(t, "//llgo:param(p) access(read)\nfunc F(p *int32) int32 { return 0 }")
	if err != nil {
		t.Fatal(err)
	}
	if err = ApplyPointerEffects(mod.Context(), mod.NamedFunction("read"), sig, attrs, 0); err != nil {
		t.Fatal(err)
	}
	optimizeValueTest(t, mod)
	if !strings.Contains(mod.NamedFunction("caller").String(), "ret i32 14") {
		t.Fatal(mod.String())
	}
}

func TestPointerAccessNativeModes(t *testing.T) {
	for mode, native := range map[string]string{"none": "readnone", "read": "readonly", "write": "writeonly", "readwrite": ""} {
		mod := valueTestModule(t, "declare void @F(ptr)")
		attrs, sig, err := parseTest(t, "//llgo:param(p) access("+mode+")\nfunc F(p *int) {}")
		if err != nil {
			t.Fatal(err)
		}
		fn := mod.NamedFunction("F")
		if err = ApplyPointerEffects(mod.Context(), fn, sig, attrs, 0); err != nil {
			t.Fatal(err)
		}
		if native != "" && fn.GetEnumAttributeAtIndex(1, llvm.AttributeKindID(native)).IsNil() {
			t.Fatal(fn.String())
		}
		if (CheckPointerEffects(fn, attrs, "test operation") != nil) != (native != "") {
			t.Fatal("unexpected instrumentation policy", mode)
		}
		if err = llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
}
