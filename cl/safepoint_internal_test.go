//go:build !llgo

package cl

import (
	"go/types"
	"testing"
)

func TestExcludeSafepointPackage(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "runtime", want: true},
		{path: "internal/runtime/atomic", want: true},
		{path: "github.com/xgo-dev/llgo/runtime", want: true},
		{path: "github.com/xgo-dev/llgo/runtime/internal/wasmevent", want: true},
		{path: "github.com/xgo-dev/llgo/runtimeextra"},
		{path: "example.com/app/runtime"},
	}
	for _, test := range tests {
		if got := excludeSafepointPackage(test.path); got != test.want {
			t.Errorf("excludeSafepointPackage(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestSafepointPackagePathUsesGenericOrigin(t *testing.T) {
	pkg := buildLinkOnceSSAPackage(t, `package p
type Box[T any] struct{}
func (Box[T]) M() {}
`)
	box := pkg.Pkg.Scope().Lookup("Box").(*types.TypeName).Type().(*types.Named)
	boxInt, err := types.Instantiate(nil, box, []types.Type{types.Typ[types.Int]}, true)
	if err != nil {
		t.Fatal(err)
	}
	_, fn := linkOnceTestMethodValue(t, pkg, boxInt, "M")
	if fn.Package() != nil || fn.Origin() == nil {
		t.Fatalf("expected a package-less generic instance, got package=%v origin=%v", fn.Package(), fn.Origin())
	}
	if got := safepointPackagePath(fn); got != "p" {
		t.Fatalf("safepointPackagePath(%s) = %q, want p", fn, got)
	}
}

func TestSafepointPackagePathLeavesSyntheticWrapperUnowned(t *testing.T) {
	pkg := buildLinkOnceSSAPackage(t, `package p
type Inner struct{}
func (Inner) M() {}
type Outer struct{ Inner }
`)
	outer := pkg.Pkg.Scope().Lookup("Outer").(*types.TypeName).Type()
	_, fn := linkOnceTestMethodValue(t, pkg, types.NewPointer(outer), "M")
	if fn.Package() != nil || fn.Origin() != nil {
		t.Fatalf("expected an unowned synthetic wrapper, got package=%v origin=%v", fn.Package(), fn.Origin())
	}
	if got := safepointPackagePath(fn); got != "" {
		t.Fatalf("safepointPackagePath(%s) = %q, want empty", fn, got)
	}
}
