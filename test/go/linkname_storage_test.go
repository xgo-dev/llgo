package gotest

import (
	"reflect"
	"testing"
	_ "unsafe"

	"github.com/xgo-dev/llgo/test/go/testdata/linknamevars/a"
	"github.com/xgo-dev/llgo/test/go/testdata/linknamevars/b"
)

// Ordinary linkname behavior is shared by gc and LLGo. Native prefixes belong
// in test/llgoext, not in this compatibility suite.
//
//go:linkname linknameStandalone llgo_test_standalone_zero
var linknameStandalone int

//go:linkname linknameExplicitZero llgo_test_explicit_zero
var linknameExplicitZero = 0

//go:linkname linknameGoBody llgo_test_go_body
func linknameGoBody(f func(int) int, n int) int { return f(n) }

//go:linkname linknameGoReference llgo_test_go_body
func linknameGoReference(func(int) int, int) int

func TestLinknameGoFunctionDefinitionAndReference(t *testing.T) {
	n := 30
	f := func(x int) int { return n + x }
	if got := linknameGoReference(f, 12); got != 42 {
		t.Fatalf("Go callback result = %d", got)
	}
}

func TestLinknameZeroDefinitions(t *testing.T) {
	if linknameStandalone != 0 || linknameExplicitZero != 0 {
		t.Fatal("standalone definitions are not zero initialized")
	}
	if &a.Zero != &b.Zero {
		t.Fatal("zero definitions have different storage")
	}
	old := a.Zero
	t.Cleanup(func() { a.Zero = old })
	a.Zero = 71
	if b.Zero != 71 {
		t.Fatalf("shared zero value = %d", b.Zero)
	}
	if &a.Empty != &b.Empty {
		t.Fatal("zero-sized linknames have different addresses")
	}
}

func TestLinknameStaticAndDynamicInitialization(t *testing.T) {
	if &a.Default != &b.Default || a.Default != 55 || a.DefaultCopied != 55 {
		t.Fatal("unannotated zero definition overrode static data")
	}
	if &a.Static != &b.Static || a.Static != 41 {
		t.Fatal("nonzero definition did not override zero storage")
	}
	if &a.Dynamic != &b.Dynamic || a.Dynamic != 18 || !reflect.DeepEqual(a.Order, []int{1, 2}) {
		t.Fatalf("dynamic value/order = %d/%v", a.Dynamic, a.Order)
	}
	if &a.Text != &b.Text || a.Text != "linked value" || a.Copied != "linked value" {
		t.Fatalf("string initialization/copy = %q/%q", a.Text, a.Copied)
	}
	if &a.Marked != &b.Marked || b.Marked != 19 {
		t.Fatal("one-argument marker lost its package symbol")
	}
	if &a.Named != &b.Named || a.Named != 23 {
		t.Fatal("qualified target lost its definition")
	}
}

func TestLinknameGoValueRepresentation(t *testing.T) {
	if &a.Closure != &b.Closure || a.Closure == nil {
		t.Fatal("closure storage is not shared")
	}
	// Invoke the initializer-created closure through both aliases before
	// replacing it, checking that its captured state is shared as well.
	if got := a.Closure(2); got != 22 {
		t.Fatalf("initialized closure = %d", got)
	}
	if got := b.Closure(-2); got != 20 {
		t.Fatalf("shared closure environment = %d", got)
	}
	original := a.Closure
	t.Cleanup(func() { a.Closure = original })
	n := 35
	a.Closure = func(x int) int { return n + x }
	if got := b.Closure(7); got != 42 {
		t.Fatalf("captured closure = %d", got)
	}
	a.Closure = nil
	if b.Closure != nil {
		t.Fatal("nil closure not shared")
	}
	if &a.Slice != &b.Slice || !reflect.DeepEqual(a.Slice, []int{2, 3, 5}) {
		t.Fatal("slice representation/initializer lost")
	}
}

//go:linkname linknameReserved go:llgo_test_reserved_storage
var linknameReserved int32

func TestLinknameGoNamespaceStorage(t *testing.T) {
	if &linknameReserved != &a.GoNamespace || linknameReserved != 0 {
		t.Fatal("go: symbol did not retain shared zero storage")
	}
	defer func() { linknameReserved = 0 }()
	linknameReserved = 42
	if a.GoNamespace != 42 {
		t.Fatal("go: symbol aliases do not share writes")
	}
}
