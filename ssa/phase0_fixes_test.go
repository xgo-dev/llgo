package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestIsPkgScopeDistinctPointers(t *testing.T) {
	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/a", "a")
	local := types.NewScope(pkgA.Scope(), token.NoPos, token.NoPos, "func")
	nested := types.NewScope(local, token.NoPos, token.NoPos, "block")

	if !isPkgScope(pkgA.Scope(), pkgA.Scope()) {
		t.Fatal("identical package scope should match")
	}
	if !isPkgScope(pkgA.Scope(), pkgB.Scope()) {
		t.Fatal("distinct package scope pointers should both be recognized as package scopes")
	}
	if isPkgScope(local, pkgA.Scope()) {
		t.Fatal("function scope must not match package scope")
	}
	if isPkgScope(nested, pkgA.Scope()) {
		t.Fatal("nested block scope must not match package scope")
	}
}
