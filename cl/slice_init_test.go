package cl

import "testing"

func TestOptionalSliceInitializers(t *testing.T) {
	pkg := buildSSAPackage(t, `package foo
var cell = &struct{ N int }{N: 17}
var Literal = []*struct{ N int }{cell, nil}
var Numbers = []int{0, 1}
var View = Numbers[:]
var Effect = []int{makeValue()}
var source = 3
var pointer = &source
var Dereference = []int{*pointer}
var Nested = [][2]int{{1, 2}}
func makeValue() int { println("effect"); return 4 }
`)
	groups := OptionalSliceInitializers(pkg.Func("init"))
	if len(groups) != 2 || groups[0].Global.Name() != "Literal" || groups[1].Global.Name() != "Numbers" {
		t.Fatalf("expected only the two independent slice literals, got %v", groups)
	}
	if len(OptionalSliceInitializers(nil)) != 0 || len(OptionalSliceInitializers(pkg.Func("makeValue"))) != 0 {
		t.Fatal("recognized a slice initializer outside of package init")
	}
}
