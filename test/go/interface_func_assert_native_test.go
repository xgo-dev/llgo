//go:build !wasm

package gotest

import "testing"

// Native-target comma-ok tuples have a separate Memory32 lowering limitation.
// See https://github.com/xgo-dev/llgo/issues/3090.
// TestNamedFunctionAssertionPanics covers single-result rejection on all targets.
func TestGoToNativeFunctionAssertionIdentity(t *testing.T) {
	for _, value := range []any{
		assertFuncA(func(x int32) int32 { return x }),
		assertFuncA(nil),
	} {
		if _, ok := value.(assertNativeFunc); ok {
			t.Fatalf("Go function %T asserted as native function", value)
		}
	}
}
