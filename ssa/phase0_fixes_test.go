package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestIsPackageLevelScope(t *testing.T) {
	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/a", "a")
	local := types.NewScope(pkgA.Scope(), token.NoPos, token.NoPos, "func")
	nested := types.NewScope(local, token.NoPos, token.NoPos, "block")

	if !isPackageLevelScope(pkgA.Scope()) {
		t.Fatal("package scope should be recognized")
	}
	if !isPackageLevelScope(pkgB.Scope()) {
		t.Fatal("distinct package scope pointer should be recognized")
	}
	if isPackageLevelScope(local) {
		t.Fatal("function scope must not match package scope")
	}
	if isPackageLevelScope(nested) {
		t.Fatal("nested block scope must not match package scope")
	}
	if isPackageLevelScope(nil) {
		t.Fatal("nil scope must not match package scope")
	}
}
