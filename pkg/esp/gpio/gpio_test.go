//go:build !llgo.idf

// Host-build tests for the gpio surface. A host build has no glue, so
// nothing here claims a pin exists: the tests pin what the pure-Go layer
// owns - the honest-stub contract (every binding reports errUnsupported
// instead of faking hardware) and the error shape those failures arrive in
// (matching by code through errors.Is, the code reachable through
// esperr.Code and errors.As, the code-era rendering of a forwarded code).
package gpio

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every binding reports errUnsupported (0x7F00) instead of faking
// a pin, and GetLevel reads low rather than inventing a level.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	p := Pin(4)
	if err := p.SetDirection(Output, PullNone); !errors.Is(err, unsupported) {
		t.Fatalf("SetDirection on a host build = %v, want errUnsupported from the stub", err)
	}
	if err := p.SetLevel(true); !errors.Is(err, unsupported) {
		t.Fatalf("SetLevel on a host build = %v, want errUnsupported", err)
	}
	if p.GetLevel() {
		t.Fatal("GetLevel on a host build = true, want the stub's low")
	}
	if err := p.Toggle(); !errors.Is(err, unsupported) {
		t.Fatalf("Toggle on a host build = %v, want errUnsupported", err)
	}
}

// TestErrorShape pins the shape of the forwarded-code errors: they match
// the same code by errors.Is (both against the sentinel-carrying esperr
// vocabulary and against a fresh literal, the code-era matching style),
// esperr.Code and errors.As reach the code, and the rendering stays the
// code-era "esp error N" (a forwarded esp_err is not translatable prose).
func TestErrorShape(t *testing.T) {
	err := Pin(4).SetLevel(true)
	if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatal("the stub error did not match the same-code Error literal")
	}
	if got := esperr.Code(err); got != errUnsupported {
		t.Fatalf("esperr.Code = %d, want 0x%X", got, errUnsupported)
	}
	var e *esperr.Error
	if !errors.As(err, &e) || e.Code != errUnsupported {
		t.Fatal("errors.As did not surface the *esperr.Error")
	}
	if got, want := err.Error(), "esp error 32512"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
