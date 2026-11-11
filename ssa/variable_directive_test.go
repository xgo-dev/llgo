// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

func TestVariableDirectivesPackageIdentity(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	makePackage := func() (*types.Package, *types.Var) {
		p := types.NewPackage("example/locations", "locations")
		v := types.NewVar(token.NoPos, p, "Line", types.Typ[types.Int32])
		p.Scope().Insert(v)
		return p, v
	}
	original, origVar := makePackage()
	alternate, altVar := makePackage()
	prog.SetPackageVariables(original, directive.Variables{"Line": {Name: "Line", Linkname: "llgo.line", Kind: directive.LineVariable}})
	prog.SetPackageVariables(alternate, directive.Variables{"Line": {Name: "Line"}})
	prog.SetLinkname("example/locations.Line", "llgo.line")
	// An alternate source view, even one with the same package path and name,
	// is authoritative. It must not inherit the original's compiler value.
	if got := prog.VariableDirective(origVar); got.Kind != directive.LineVariable {
		t.Fatalf("original: %+v", got)
	}
	if got := prog.VariableDirective(altVar); got.Kind != directive.OrdinaryVariable {
		t.Fatalf("alternate: %+v", got)
	}
	local := types.NewVar(token.NoPos, original, "Line", types.Typ[types.Int32])
	if got := prog.VariableDirective(local); got.Kind != directive.OrdinaryVariable {
		t.Fatalf("local inherited global directive: %+v", got)
	}
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	if got := backend.VariableDirective(origVar); got.Kind != directive.LineVariable {
		t.Fatalf("backend lost prepared metadata: %+v", got)
	}
	consumer := types.NewPackage("example/consumer", "consumer")
	consumer.SetImports([]*types.Package{original})
	alternateConsumer := types.NewPackage("example/consumer", "consumer")
	alternateConsumer.SetImports([]*types.Package{alternate})
	transitive := types.NewPackage("example/transitive", "transitive")
	transitive.SetImports([]*types.Package{consumer})
	for _, test := range []struct {
		pkg  *types.Package
		want bool
	}{{original, true}, {alternate, false}, {consumer, true}, {alternateConsumer, false}, {transitive, false}} {
		if got := backend.MayUseCompilerVariables(test.pkg); got != test.want {
			t.Errorf("MayUseCompilerVariables(%s) = %v, want %v", test.pkg.Path(), got, test.want)
		}
	}
	// Effective patch publication can remove the last compiler variable.
	prog.SetPackageVariables(original, directive.Variables{"Line": {Name: "Line"}})
	if prog.MayUseCompilerVariables(consumer) {
		t.Fatal("consumer retained stale compiler-variable presence after replacement")
	}
}
