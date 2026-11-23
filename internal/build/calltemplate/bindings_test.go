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
		"func Bytes(v any) ([]byte, error) { return nil, nil }\n"
	const bind = "\n//llgointernal:template:bind "
	const renderer = "Render gofmt format=pattern args=values composite=true"
	const binary = "WireSize binary op=size value=v"
	for _, tc := range []struct {
		name, comment, errorText string
	}{
		{name: "named entries", comment: bind + renderer + bind + binary},
		{name: "unknown function", comment: bind + "Missing gofmt", errorText: "invalid template entry"},
		{name: "unknown dialect", comment: bind + "Render unknown", errorText: "unknown template dialect"},
		{name: "unknown operation", comment: bind + "WireSize binary op=marshal", errorText: "unknown binary template operation"},
		{name: "conflicting codecs", comment: bind + binary + bind + "Bytes json op=marshal value=v", errorText: "conflicting codec binding"},
		{name: "duplicate codec", comment: bind + binary + bind + binary, errorText: "conflicting codec binding"},
		{name: "duplicate formatter", comment: bind + renderer + bind + renderer, errorText: "duplicate Go format binding"},
		{name: "duplicate composite", comment: bind + renderer + bind + "RenderOther gofmt format=pattern args=values composite=true", errorText: "duplicate value format binding"},
		{name: "bad codec role", comment: bind + "WireSize binary op=size value=x", errorText: "invalid value role"},
		{name: "bad codec arity", comment: bind + "WireSize binary op=size value=v buffer=v", errorText: "invalid signature or roles"},
		{name: "bad codec result", comment: bind + "Bytes binary op=size value=v", errorText: "invalid signature or roles"},
		{name: "bad gofmt option", comment: bind + "Render gofmt format=pattern args=values composite=false", errorText: "invalid composite format binding"},
		{name: "bad argument name", comment: bind + "Render gofmt format=unknown args=values", errorText: "unknown format parameter"},
		{name: "duplicate operand", comment: bind + "Render gofmt format=pattern args=values args=values", errorText: "invalid template binding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := checkedSource(t, source+tc.comment)
			roots := []*packages.Package{pkg}
			untrusted, err := LoadBindings(roots, func(*packages.Package) bool { return false })
			if err != nil || untrusted.NeedsFormatRuntime() || len(untrusted.codecs) != 0 {
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
			if err != nil || !got.NeedsFormatRuntime() || len(got.codecs) != 1 || got.formats[render].format != 0 ||
				got.formats[render].result != "string" || got.codecs[0].entries["WireSize"] != "size" {
				t.Fatalf("trusted bindings = %+v, %v", got, err)
			}
		})
	}
}
