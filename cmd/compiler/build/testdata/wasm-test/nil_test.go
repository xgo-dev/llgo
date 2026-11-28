package wasmtest

import (
	"fmt"
	"strings"
	"testing"
)

type nilGuardStruct struct {
	padding [4096]byte
	value   int
}

//go:noinline
func nilGuardPointer() *int { return nil }

//go:noinline
func nilGuardField(p *nilGuardStruct) int { return p.value }

//go:noinline
func nilGuardStore(p *int) { *p = 17 }

//go:noinline
func nilGuardArray(p *[2]int, i int) int { return p[i] }

//go:noinline
func nilGuardCall(f func() int) int { return f() }

//go:noinline
func nilGuardDeferredCall(f func()) { defer f() }

//go:noinline
func nilGuardStoreOperandOrder(p *nilGuardStruct, value func() int) {
	p.value = value()
}

func TestNilStoreOperandOrder(t *testing.T) {
	called := false
	defer func() {
		if recover() == nil {
			t.Error("nil field store did not panic")
		}
		if !called {
			t.Error("nil store panicked before evaluating its right-hand operand")
		}
	}()
	nilGuardStoreOperandOrder(nil, func() int { called = true; return 1 })
}

func TestNilPointerAndFunctionRecovery(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func()
	}{
		{"dynamic load", func() { t.Errorf("loaded nil pointer: %d", *nilGuardPointer()) }},
		{"field beyond guard page", func() { _ = nilGuardField(nil) }},
		{"store", func() { nilGuardStore(nilGuardPointer()) }},
		{"array", func() { _ = nilGuardArray(nil, 1) }},
		{"function", func() { _ = nilGuardCall(nil) }},
		{"deferred function", func() { nilGuardDeferredCall(nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(fmt.Sprint(r), "nil pointer dereference") {
					t.Errorf("recover = %v, want a Go nil-pointer panic", r)
				}
			}()
			tc.call()
		})
	}
	var value int
	nilGuardStore(&value)
	if value != 17 || nilGuardCall(func() int { return value }) != 17 {
		t.Fatal("non-nil operation changed")
	}
}
