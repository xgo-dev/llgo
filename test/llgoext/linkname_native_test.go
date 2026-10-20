//go:build llgo && !wasm

package llgoext

import (
	"testing"
	"unsafe"
)

func TestLinknameNativeVariables(t *testing.T) {
	if nativeAddresses(unsafe.Pointer(&nativeCounter), unsafe.Pointer(&nativeArray), unsafe.Pointer(&nativePointer), unsafe.Pointer(&nativeCallback)) != 1 {
		t.Fatal("Go bindings do not reference native storage")
	}
	if &nativeCounterAlias != &nativeCounter || nativeCounter != 41 || nativeArray != [3]int32{2, 3, 5} || nativePointer != &nativeCounter {
		t.Fatal("native initializers or alias identity lost")
	}
	oldCounter, oldCallback := nativeCounter, nativeCallback
	t.Cleanup(func() { nativeCounter = oldCounter; nativeCallback = oldCallback })
	nativeCounterAlias = 17
	if *nativePointer != 17 {
		t.Fatal("native write not visible through pointer")
	}
	if got := nativeCallback(21); got != 42 {
		t.Fatalf("native callback = %d", got)
	}
	nativeCallback = nil
	if nativeCallback != nil {
		t.Fatal("nil native function pointer not preserved")
	}
}

func TestLinknameNativeFunctionDefinitions(t *testing.T) {
	if got := nativeGoPlaceholder(33); got != 42 {
		t.Fatalf("Go ABI placeholder binding = %d", got)
	}
	for name, call := range map[string]func(int32) int32{
		"Go body": nativeEntry, "C entry": invokeNativeEntry,
		"foreign thread": invokeNativeThread, "llgo placeholder": nativePlaceholder,
	} {
		t.Run(name, func(t *testing.T) {
			if got := call(35); got != 42 {
				t.Fatalf("entry result = %d", got)
			}
		})
	}
	if got := invokeNativeCallbackEntry(20); got != 41 {
		t.Fatalf("native-to-Go callback argument = %d", got)
	}
}
