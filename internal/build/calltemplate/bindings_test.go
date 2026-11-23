package calltemplate

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestLibraryTemplateBindings(t *testing.T) {
	const render = "func Render(pattern string, values ...any) string { return pattern }\n"
	const other = "func RenderOther(pattern string, values ...any) string { return pattern }\n"
	const bind = "//llgointernal:template:gofmt "
	const composite = bind + "format=pattern args=values composite=true\n"
	for _, tc := range []struct {
		name, declarations, errorText string
	}{
		{"attached format", composite + render, ""},
		{"duplicate formatter", composite + composite + render, "duplicate Go format binding"},
		{"duplicate composite", composite + render + composite + other, "duplicate value format binding"},
		{"bad gofmt option", bind + "format=pattern args=values composite=false\n" + render, "invalid composite format binding"},
		{"bad argument name", bind + "format=unknown args=values\n" + render, "unknown format parameter"},
		{"duplicate operand", bind + "format=pattern args=values args=values\n" + render, "invalid template binding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := checkedSource(t, "package fmt\n"+tc.declarations)
			pkg.PkgPath = "_llgo_fmt"
			roots := []*packages.Package{pkg}
			untrusted, err := LoadBindings(roots, func(*packages.Package) bool { return false })
			if err != nil || untrusted.NeedsFormatRuntime() {
				t.Fatalf("untrusted bindings = %+v, %v", untrusted, err)
			}
			got, err := LoadBindings(roots, func(*packages.Package) bool { return true })
			if tc.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorText) {
					t.Fatalf("error = %v, want %q", err, tc.errorText)
				}
				return
			}
			fn := pkg.Types.Scope().Lookup("Render").(*types.Func)
			if err != nil || !got.NeedsFormatRuntime() || got.formats[fn].format != 0 || got.formats[fn].result != "string" {
				t.Fatalf("trusted bindings = %+v, %v", got, err)
			}
		})
	}
}
