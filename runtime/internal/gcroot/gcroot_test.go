package gcroot

import (
	"testing"
	"unsafe"
)

type testFrameMap struct {
	frameMap
	meta [1]unsafe.Pointer
}

type testStackEntry struct {
	stackEntry
	roots [2]unsafe.Pointer
}

func TestVisitAndSwitchContexts(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	meta := unsafe.Pointer(uintptr(0x33))
	firstValue := unsafe.Pointer(uintptr(0x11))
	secondValue := unsafe.Pointer(uintptr(0x22))
	m := testFrameMap{
		frameMap: frameMap{numRoots: 2, numMeta: 1},
		meta:     [1]unsafe.Pointer{meta},
	}
	entry := testStackEntry{
		stackEntry: stackEntry{m: &m.frameMap},
		roots:      [2]unsafe.Pointer{firstValue, secondValue},
	}
	currentRootChain = uintptr(unsafe.Pointer(&entry.stackEntry))

	var first, second Context
	RegisterActive(&first)
	Register(&second)

	var values, metadata []unsafe.Pointer
	Visit(func(root *unsafe.Pointer, meta unsafe.Pointer) {
		values = append(values, *root)
		metadata = append(metadata, meta)
	})
	if len(values) != 2 || values[0] != firstValue || values[1] != secondValue {
		t.Fatalf("Visit values = %v, want [%p %p]", values, firstValue, secondValue)
	}
	if metadata[0] != meta || metadata[1] != nil {
		t.Fatalf("Visit metadata = %v, want [%p nil]", metadata, meta)
	}

	Switch(&second)
	if first.chain != unsafe.Pointer(&entry.stackEntry) || currentRootChain != 0 {
		t.Fatal("Switch did not save the active chain and restore the next chain")
	}
	Unregister(&first)
	if contexts != &second || second.next != nil {
		t.Fatal("Unregister did not unlink the suspended context")
	}
}

func TestRejectsInvalidContextOperations(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	var ctx Context
	assertPanics(t, func() { Register(nil) })
	RegisterActive(&ctx)
	assertPanics(t, func() { Register(&ctx) })
	assertPanics(t, func() { RegisterActive(new(Context)) })
	assertPanics(t, func() { Switch(nil) })
	assertPanics(t, func() { Unregister(&ctx) })
}

func TestRejectsInvalidFrameMap(t *testing.T) {
	m := frameMap{numRoots: 1, numMeta: 2}
	entry := stackEntry{m: &m}
	assertPanics(t, func() {
		visitChain(unsafe.Pointer(&entry), func(*unsafe.Pointer, unsafe.Pointer) {})
	})
}

func TestRestoreChain(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	first := unsafe.Pointer(uintptr(0x11))
	second := unsafe.Pointer(uintptr(0x22))
	currentRootChain = uintptr(first)
	if got := CurrentChain(); got != first {
		t.Fatalf("CurrentChain() = %p, want %p", got, first)
	}
	RestoreChain(second)
	if got := CurrentChain(); got != second {
		t.Fatalf("CurrentChain() after restore = %p, want %p", got, second)
	}
}

func TestAdoptCurrent(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	var first, second Context
	RegisterActive(&first)
	Register(&second)
	currentRootChain = uintptr(0x11)

	AdoptCurrent(&second)
	if activeContext != uintptr(unsafe.Pointer(&second)) {
		t.Fatal("AdoptCurrent did not replace the active context")
	}
	if currentRootChain != uintptr(0x11) {
		t.Fatal("AdoptCurrent changed the chain restored by the stack switch")
	}
}

func TestPublishAndSwitchToSystem(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	var ctx Context
	RegisterActive(&ctx)
	currentRootChain = uintptr(0x44)
	PublishCurrent()
	if ctx.chain != unsafe.Pointer(uintptr(0x44)) {
		t.Fatalf("published chain = %p, want %p", ctx.chain, unsafe.Pointer(uintptr(0x44)))
	}

	SwitchAtBoundary(nil)
	if activeContext != 0 || currentRootChain != 0 {
		t.Fatalf("system boundary retained active=%#x chain=%#x", activeContext, currentRootChain)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	fn()
}

func resetForTest() {
	contexts = nil
	currentRootChain = 0
	activeContext = 0
}
