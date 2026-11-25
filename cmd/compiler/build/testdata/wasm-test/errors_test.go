package wasmtest

import (
	"errors"
	"fmt"
	"sort"
	"testing"
)

type wasmError struct{}

func (*wasmError) Error() string { return "wasm error" }
func (*wasmError) Code() int     { return 7 }

type codedError interface {
	error
	Code() int
}

func TestErrorsAsTargets(t *testing.T) {
	// errors.As calls reflectlite.Type.Kind through an interface. Its return
	// type must agree with Go's uint8 Kind, including on Memory64 targets.
	want := &wasmError{}
	err := fmt.Errorf("wrapped: %w", want)
	var ptr *wasmError
	if !errors.As(err, &ptr) || ptr != want {
		t.Fatalf("errors.As pointer target = %v, want %v", ptr, want)
	}
	var iface codedError
	if !errors.As(err, &iface) || iface != want || iface.Code() != 7 {
		t.Fatalf("errors.As interface target = %v, want %v", iface, want)
	}

	// Invalid targets must produce the ordinary recoverable Go panic, not a
	// host trap from a mismatched indirect-call signature.
	var notError int
	for _, target := range []any{42, (*wasmError)(nil), &notError} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("errors.As accepted invalid target %T", target)
				}
			}()
			errors.As(err, target)
		}()
	}
}

func TestReflectliteSwapperKinds(t *testing.T) {
	// sort.Slice uses reflectlite.Swapper, including its string fast path.
	values := []string{"charlie", "alpha", "bravo"}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	if values[0] != "alpha" || values[1] != "bravo" || values[2] != "charlie" {
		t.Fatalf("sort.Slice = %v", values)
	}

	// Formatting ValueError also crosses the Kind-to-runtime-Kind boundary.
	defer func() {
		if got := fmt.Sprint(recover()); got != "reflect: call of Swapper on int Value" {
			t.Fatalf("sort.Slice panic = %q", got)
		}
	}()
	sort.Slice(42, func(i, j int) bool { return i < j })
}
