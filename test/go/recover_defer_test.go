package gotest

import (
	"reflect"
	"testing"
)

func recoverIndirect() any {
	return recover()
}

func recoverRecursive(n int) any {
	if n == 0 {
		return recoverRecursive(1)
	}
	return recover()
}

func TestRecoverOnlyDirectDeferredCall(t *testing.T) {
	var indirect, direct, second any
	func() {
		defer func() {
			indirect = recoverIndirect()
			direct = recover()
			second = recover()
		}()
		panic("direct-sentinel")
	}()

	if indirect != nil {
		t.Fatalf("indirect recover = %v, want nil", indirect)
	}
	if direct != "direct-sentinel" {
		t.Fatalf("direct recover = %v, want direct-sentinel", direct)
	}
	if second != nil {
		t.Fatalf("second recover = %v, want nil", second)
	}
}

func TestRecoverRejectsRecursiveIndirectCall(t *testing.T) {
	var indirect, direct any
	func() {
		defer func() {
			indirect = recoverRecursive(0)
			direct = recover()
		}()
		panic("recursive-sentinel")
	}()

	if indirect != nil {
		t.Fatalf("recursive indirect recover = %v, want nil", indirect)
	}
	if direct != "recursive-sentinel" {
		t.Fatalf("direct recover = %v, want recursive-sentinel", direct)
	}
}

func TestNestedPanicRecoverStack(t *testing.T) {
	var recovered []any
	func() {
		defer func() {
			recovered = append(recovered, recover())
		}()
		defer func() {
			defer func() {
				recovered = append(recovered, recover())
			}()
			panic("inner")
		}()
		panic("outer")
	}()

	want := []any{"inner", "outer"}
	if !reflect.DeepEqual(recovered, want) {
		t.Fatalf("recover stack = %v, want %v", recovered, want)
	}
}

func TestDeferredRecoverBuiltinKeepsNestedPanicForNextDefer(t *testing.T) {
	var recovered []any
	func() {
		defer func() {
			recovered = append(recovered, recover())
		}()
		defer func() {
			defer func() {
				recovered = append(recovered, recover())
			}()
			defer recover()
			panic("inner")
		}()
		panic("outer")
	}()

	want := []any{"inner", "outer"}
	if !reflect.DeepEqual(recovered, want) {
		t.Fatalf("recover stack after deferred recover builtin = %v, want %v", recovered, want)
	}
}

func TestDeferredRecoverBuiltinCanRecoverOuterPanicAfterNestedRecover(t *testing.T) {
	var recovered []any
	func() {
		defer func() {
			recovered = append(recovered, recover())
		}()
		defer func() {
			defer recover()
			defer func() {
				recovered = append(recovered, recover())
			}()
			panic("inner")
		}()
		panic("outer")
	}()

	want := []any{"inner", nil}
	if !reflect.DeepEqual(recovered, want) {
		t.Fatalf("recover stack after outer deferred recover builtin = %v, want %v", recovered, want)
	}
}

type recoverValueMethod uintptr

var methodWrapperRecovered any

func (recoverValueMethod) recoverViaValueMethod() {
	methodWrapperRecovered = recover()
}

func TestRecoverThroughDeferredPointerToValueMethodWrapper(t *testing.T) {
	methodWrapperRecovered = nil
	var x recoverValueMethod
	func() {
		defer (*recoverValueMethod).recoverViaValueMethod(&x)
		panic("method-wrapper-sentinel")
	}()

	if methodWrapperRecovered != "method-wrapper-sentinel" {
		t.Fatalf("method wrapper recover = %v, want method-wrapper-sentinel", methodWrapperRecovered)
	}
}

func TestRecoverThroughMethodWrapperStillRequiresDirectDeferredCall(t *testing.T) {
	methodWrapperRecovered = "unset"
	var direct any
	var x recoverValueMethod
	func() {
		defer func() {
			(*recoverValueMethod).recoverViaValueMethod(&x)
			direct = recover()
		}()
		panic("outer-sentinel")
	}()

	if methodWrapperRecovered != nil {
		t.Fatalf("nested method wrapper recover = %v, want nil", methodWrapperRecovered)
	}
	if direct != "outer-sentinel" {
		t.Fatalf("direct recover after nested method wrapper = %v, want outer-sentinel", direct)
	}
}
