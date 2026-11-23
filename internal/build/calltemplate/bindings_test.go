package calltemplate

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestLibraryTemplateBindings(t *testing.T) {
	const source = "package patched\n" +
		"func Render(pattern string, values ...any) string { return pattern }\n" +
		"func RenderOther(pattern string, values ...any) string { return pattern }\n" +
		"func WireSize(v any) int { return 0 }\n" +
		"func Bytes(v any) ([]byte, error) { return nil, nil }\n" +
		"func wireSize[T any](v T) int { return 0 }; const body = \"return 0\"\n"
	const bind = "\n//llgointernal:template:bind "
	const renderer = "Render gofmt format=pattern args=values composite=true"
	const expansion = "WireSize any helper=wireSize body=body"
	for _, tc := range []struct {
		name, comment, errorText string
	}{
		{name: "named entries", comment: bind + renderer + bind + expansion},
		{name: "unknown function", comment: bind + "Missing gofmt", errorText: "invalid template entry"},
		{name: "unknown dialect", comment: bind + "Render unknown", errorText: "unknown template dialect"},
		{name: "unknown any option", comment: bind + "WireSize any op=size", errorText: "unknown any template option"},
		{name: "invalid template constant", comment: bind + "WireSize any helper=wireSize body=Missing", errorText: "must be a string constant"},
		{name: "duplicate any template", comment: bind + expansion + bind + expansion, errorText: "duplicate any template"},
		{name: "duplicate formatter", comment: bind + renderer + bind + renderer, errorText: "duplicate Go format binding"},
		{name: "duplicate composite", comment: bind + renderer + bind + "RenderOther gofmt format=pattern args=values composite=true", errorText: "duplicate value format binding"},
		{name: "bad result", comment: bind + "Bytes any helper=wireSize body=body", errorText: "incompatible signature"},
		{name: "bad gofmt option", comment: bind + "Render gofmt format=pattern args=values composite=false", errorText: "invalid composite format binding"},
		{name: "bad argument name", comment: bind + "Render gofmt format=unknown args=values", errorText: "unknown format parameter"},
		{name: "duplicate operand", comment: bind + "Render gofmt format=pattern args=values args=values", errorText: "invalid template binding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := checkedSource(t, source+tc.comment)
			roots := []*packages.Package{pkg}
			untrusted, err := LoadBindings(roots, func(*packages.Package) bool { return false })
			if err != nil || untrusted.NeedsFormatRuntime() || len(untrusted.structural) != 0 {
				t.Fatalf("untrusted bindings = %+v, %v", untrusted, err)
			}
			got, err := LoadBindings(roots, func(*packages.Package) bool { return true })
			if tc.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorText) {
					t.Fatalf("error = %v, want %q", err, tc.errorText)
				}
				return
			}
			render := pkg.Types.Scope().Lookup("Render").(*types.Func)
			if err != nil || !got.NeedsFormatRuntime() || len(got.structural) != 1 || got.formats[render].format != 0 ||
				got.formats[render].result != "string" || got.structural[0].entries["WireSize"].helper.Name() != "wireSize" {
				t.Fatalf("trusted bindings = %+v, %v", got, err)
			}
		})
	}
}
