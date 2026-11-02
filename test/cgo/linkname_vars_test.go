//go:build llgo && !wasm

package cgo

import "testing"

func TestLinknameCVariables(t *testing.T) {
	if !linkedVarAddressesMatch() {
		t.Fatal("C-prefixed bindings do not share the native variables' addresses")
	}
	if &linkedScalarAlias != &linkedScalar {
		t.Fatal("llgo:link and go:linkname aliases do not share the same variable address")
	}
	if linkedScalar != 42 || linkedScalarAlias != 42 {
		t.Fatalf("native scalar initializer lost: %d, %d", linkedScalar, linkedScalarAlias)
	}
	if linkedArray != [3]int32{3, 5, 7} {
		t.Fatalf("native array initializer lost: %v", linkedArray)
	}
	if linkedPointer != &linkedScalar {
		t.Fatal("native pointer does not point to the bound scalar")
	}
	t.Cleanup(func() {
		linkedScalar = 42
		linkedArray[1] = 5
	})
	*linkedPointer = 91
	if linkedScalarAlias != 91 || linkedVarReadScalar() != 91 {
		t.Fatal("Go writes are not visible through the alias and C")
	}
	linkedVarWriteArray(11)
	if linkedArray[1] != 11 {
		t.Fatal("C writes are not visible through the array binding")
	}
	if linkedCallback == nil {
		t.Fatal("native callback initializer lost")
	}
	if got := linkedCallback(21); got != 42 {
		t.Fatalf("native function pointer returned %d, want 42", got)
	}
	callback := linkedCallback
	linkedCallback = nil
	if linkedCallback != nil {
		t.Fatal("native function pointer could not be cleared")
	}
	linkedCallback = callback
	if got := linkedVarCall(23); got != 46 {
		t.Fatalf("restored function pointer returned %d through C, want 46", got)
	}
}

func TestLinknameCVariableCallbackStore(t *testing.T) {
	resetLinkedCallbacks()
	t.Cleanup(resetLinkedCallbacks)
	// Each native slot is one word. Adjacent slots make an oversized Go
	// funcval store fail regardless of the linker's global variable order.
	saved := linkedCallbacks[0]
	restore := func() { linkedCallbacks[0] = saved }
	linkedCallbacks[0] = nil
	if linkedCallbacks[1] == nil {
		t.Fatal("clearing a native callback overwrote the adjacent slot")
	}
	restore()
	for i, callback := range linkedCallbacks {
		if callback == nil {
			t.Fatalf("restoring a native callback overwrote slot %d", i)
		}
		if got := callback(21); got != 42 {
			t.Fatalf("callback slot %d returned %d, want 42", i, got)
		}
	}
}

func TestLinknameCVariableCallbackRejectsClosure(t *testing.T) {
	resetLinkedCallbacks()
	t.Cleanup(resetLinkedCallbacks)
	value := int32(7)
	defer func() {
		if recover() == nil {
			t.Error("capturing Go closure accepted in a native callback slot")
		}
		if got := linkedCallbacks[0](21); got != 42 {
			t.Errorf("rejected assignment changed the native callback: %d", got)
		}
	}()
	linkedCallbacks[0] = func(v int32) int32 { return v + value }
}

func TestLinknameCFunctionPointerStorage(t *testing.T) {
	callback := linkedCallbackBox.Callback
	t.Cleanup(func() {
		linkedCallbackBox.Callback = callback
	})
	linkedCallbackBox.Callback = nil
	if linkedCallbackBox.Callback != nil {
		t.Fatal("native function pointer could not be cleared")
	}
	if got := linkedCallbackBox.Guard; got != 0xfeedface {
		t.Fatalf("clearing a native callback overwrote adjacent storage: %#x", got)
	}
	linkedCallbackBox.Callback = callback
	if got := linkedCallbackBox.Callback(21); got != 42 {
		t.Fatalf("restored native function pointer returned %d, want 42", got)
	}
	if got := linkedCallbackBox.Guard; got != 0xfeedface {
		t.Fatalf("restoring a native callback overwrote adjacent storage: %#x", got)
	}
}
