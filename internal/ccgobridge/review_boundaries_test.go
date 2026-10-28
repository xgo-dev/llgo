package ccgobridge

import (
	"go/format"
	"strings"
	"testing"
)

func TestRewriteLeavesUnqualifiedFunctionLoadsUntouched(t *testing.T) {
	for _, body := range []string{
		`func load(p *func()) func() { return *p }`,
		`type F func(); func pointer(p *F) *F { return p }; func load(p *F) F { return *pointer(p) }`,
		`func load(p unsafe.Pointer) func() { return *(*func())(p) }`,
		`func load(p uintptr) func() { return *(*func())(unsafe.Pointer(p)) }`,
		`func load(p uintptr) func() { return *(*func())(unsafe.Pointer(&p)) }`,
		`func load(p uintptr) func() { return *(*func())(unsafe.Pointer(&struct{ value uintptr }{p})) }`,
		`func load(p uintptr) func() { return *(*func())(unsafe.Pointer(&struct{ uintptr; n int }{p, 0})) }`,
		`func load(p uintptr) func() { return *(*func())(unsafe.Pointer(&struct{ uintptr }{uintptr:p})) }`,
	} {
		t.Run(body, func(t *testing.T) {
			source := "package fixture\nimport \"unsafe\"\nvar _ unsafe.Pointer\n" + body
			before, err := format.Source([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			after, stats, err := rewriteSource(t, source)
			if err != nil || stats != (Stats{}) || after != string(before) {
				t.Fatalf("unqualified consumer changed: %+v, %v\n%s", stats, err, after)
			}
		})
	}
}

func TestRewriteRejectsReservedNamesAndInvalidSlots(t *testing.T) {
	for _, test := range []struct{ source, message string }{
		{`var __llgo_ccgo_user = 1`, "generated identifier"},
		{`func __ccgo_abiInternal_bad() {}`, "callback slot shape"},
	} {
		source := "package fixture\n" + test.source
		before, err := format.Source([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		after, stats, err := rewriteSource(t, source)
		if err == nil || !strings.Contains(err.Error(), test.message) || stats != (Stats{}) || after != string(before) {
			t.Fatalf("rejected input mutated: %+v, %v\n%s", stats, err, after)
		}
	}
}

func TestStaticExportRetainsImportedParameterTypes(t *testing.T) {
	source := `package fixture
import "unsafe"
import clock "time"
import locking "sync"
var _ locking.Once
` + exportHelper + `
func callback(v clock.Duration) clock.Duration { return v }
var pointer = __ccgo_fp(callback)
`
	out, stats, err := rewriteSource(t, source)
	if err != nil || stats.StaticBoxes != 1 || stats.Exports != 1 ||
		!strings.Contains(out, "func(v clock.Duration) clock.Duration") ||
		!strings.Contains(out, "once locking.Once") {
		t.Fatalf("typed export imports: %+v, %v\n%s", stats, err, out)
	}
}
