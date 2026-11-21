package cl

import (
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestStaticPointerCopies(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		folded bool
	}{
		{"same backing object", `
var First = base
var Second = First
`, true},
		{"initializer can replace the pointer", `
var changed = replace()
var First = base
var Second = First
func replace() int {
	base = &record{2}
	return 0
}
`, false},
		{"pointer address escapes before the copy", `
var ref = &base
var First = base
var Second = First
`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildSSAPackage(t, `package foo
type record struct { value int }
var base = &record{1}
`+test.source)
			p := newMapInitContext(t)
			p.collectStaticGlobalInits(pkg)
			base := pkg.Members["base"].(*ssa.Global)
			original, ok := p.staticGlobalInits[base]
			if !ok {
				t.Fatal("base pointer was not statically initialized")
			}
			for _, name := range []string{"First", "Second"} {
				alias := pkg.Members[name].(*ssa.Global)
				got, folded := p.staticGlobalInits[alias]
				if folded != test.folded {
					t.Errorf("static copy of %s = %v, want %v", name, folded, test.folded)
				}
				if folded && got != original {
					t.Errorf("%s points to a different backing object", name)
				}
			}
		})
	}
}
