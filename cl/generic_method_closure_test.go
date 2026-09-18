//go:build !llgo

package cl

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa/ssautil"
)

// Regression for golang/go#81135, encountered while compiling TypeScript's SyncSet.Keys.
func TestGenericMethodNestedClosureTypeArgs(t *testing.T) {
	ssapkg := buildSSAPackage(t, `package foo
type Set[T comparable] struct { values []T }
func (s *Set[T]) Keys() func(func(T) bool) {
	return func(yield func(T) bool) {
		visit := func(value T) bool { return yield(value) }
		for _, value := range s.values {
			if !visit(value) { return }
		}
	}
}
func (s *Set[T]) Visit() {
	for value := range s.Keys() {
		func() { _ = value }()
	}
}
func use() {
	new(Set[int]).Visit()
	new(Set[string]).Visit()
}
`)
	counts := make(map[types.BasicKind]int)
	for fn := range ssautil.AllFunctions(ssapkg.Prog) {
		if fn.Parent() == nil || fn.Pkg != nil {
			continue
		}
		// x/tools names the nested closures by their lexical nesting depth.
		// Check only the two generic closure bodies targeted by this regression.
		if fn.Name() != "Keys$1$1" && fn.Name() != "Visit$1$1" {
			continue
		}
		args := fn.TypeArgs()
		if len(args) != 1 {
			t.Fatalf("%s: type arguments = %v, want receiver type argument", fn, args)
		}
		if fn.Origin() == nil {
			t.Fatalf("%s: missing generic origin", fn)
		}
		counts[args[0].(*types.Basic).Kind()]++
	}
	for _, kind := range []types.BasicKind{types.Int, types.String} {
		if counts[kind] != 2 {
			t.Errorf("%s: got %d nested closures, want 2", types.Typ[kind], counts[kind])
		}
	}
}
