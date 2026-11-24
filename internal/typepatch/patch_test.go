package typepatch

import (
	"go/token"
	"go/types"
	"testing"
)

func TestCloneAndMerge(t *testing.T) {
	orig := types.NewPackage("example.com/pkg", "pkg")
	origKeep := types.NewTypeName(token.NoPos, orig, "Keep", types.Typ[types.Int])
	origOverride := types.NewTypeName(token.NoPos, orig, "Override", types.Typ[types.Int])
	origSkip := types.NewTypeName(token.NoPos, orig, "Skip", types.Typ[types.Int])
	orig.Scope().Insert(origKeep)
	orig.Scope().Insert(origOverride)
	orig.Scope().Insert(origSkip)
	orig.MarkComplete()

	alt := types.NewPackage("example.com/pkg", "pkg")
	altOverride := types.NewTypeName(token.NoPos, alt, "Override", types.Typ[types.String])
	alt.Scope().Insert(altOverride)
	alt.MarkComplete()

	cloned := Clone(alt)
	if cloned == alt || cloned.Scope() == alt.Scope() {
		t.Fatal("Clone must return an independent package and scope")
	}
	if !cloned.Complete() {
		t.Fatal("Clone did not preserve Complete")
	}
	if IsPatched(orig) || IsPatched(nil) {
		t.Fatal("unexpected IsPatched before Merge")
	}

	Merge(cloned, orig, map[string]struct{}{"Skip": {}}, false)
	if !IsPatched(orig) {
		t.Fatal("orig not marked patched after Merge")
	}
	if got := cloned.Scope().Lookup("Keep"); got != origKeep {
		t.Fatalf("Keep = %v, want %v", got, origKeep)
	}
	if origKeep.Parent() != orig.Scope() {
		t.Fatal("Merge mutated origKeep.Parent")
	}
	if got := cloned.Scope().Lookup("Override"); got != altOverride {
		t.Fatalf("Override = %v, want altOverride", got)
	}
	if got := cloned.Scope().Lookup("Skip"); got != nil {
		t.Fatalf("Skip unexpectedly copied: %v", got)
	}
	if got := alt.Scope().Lookup("Keep"); got != nil {
		t.Fatalf("Merge mutated original alt scope: %v", got)
	}
}

func TestMergeSkipAll(t *testing.T) {
	orig := types.NewPackage("example.com/skipall", "skipall")
	orig.Scope().Insert(types.NewTypeName(token.NoPos, orig, "X", types.Typ[types.Int]))
	alt := Clone(types.NewPackage("example.com/skipall", "skipall"))

	Merge(alt, orig, nil, true)
	if !IsPatched(orig) {
		t.Fatal("orig not marked patched on skipall")
	}
	if alt.Scope().Lookup("X") != nil {
		t.Fatal("skipall Merge unexpectedly copied X")
	}
}
