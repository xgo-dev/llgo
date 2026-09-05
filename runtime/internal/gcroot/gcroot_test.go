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
	currentRootChain = unsafe.Pointer(&entry.stackEntry)

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
	if first.chain != unsafe.Pointer(&entry.stackEntry) || currentRootChain != nil {
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
	currentRootChain = first
	if currentRootChain != first {
		t.Fatalf("current root chain = %p, want %p", currentRootChain, first)
	}
	RestoreChain(second)
	if currentRootChain != second {
		t.Fatalf("current root chain after restore = %p, want %p", currentRootChain, second)
	}
}

func TestAdoptCurrent(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	var first, second Context
	RegisterActive(&first)
	Register(&second)
	currentRootChain = unsafe.Pointer(uintptr(0x11))

	AdoptCurrent(&second)
	if active != &second {
		t.Fatal("AdoptCurrent did not replace the active context")
	}
	if currentRootChain != unsafe.Pointer(uintptr(0x11)) {
		t.Fatal("AdoptCurrent changed the chain restored by the stack switch")
	}
}

func TestBeginAndFinishRebuild(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	firstChain := unsafe.Pointer(uintptr(0x11))
	staleSecondChain := unsafe.Pointer(uintptr(0x22))
	var first, second Context
	currentRootChain = firstChain
	RegisterActive(&first)
	Register(&second)
	second.chain = staleSecondChain

	BeginRebuild(&second)
	if first.chain != firstChain {
		t.Fatalf("source chain = %p, want %p", first.chain, firstChain)
	}
	if active != &second || second.chain != nil || currentRootChain != nil {
		t.Fatal("BeginRebuild did not switch ownership and clear stale root links")
	}
	if !Rebuilding() {
		t.Fatal("BeginRebuild did not mark the replay active")
	}

	rebuiltSecondChain := unsafe.Pointer(uintptr(0x33))
	currentRootChain = rebuiltSecondChain
	FinishRebuild()
	if Rebuilding() {
		t.Fatal("FinishRebuild left the replay active")
	}

	SwitchAtBoundary(&first)
	if second.chain != rebuiltSecondChain || currentRootChain != firstChain {
		t.Fatal("rebuilt root links were not preserved by the next switch")
	}
}

func TestBeginAndFinishSJLJReplay(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	chain := unsafe.Pointer(uintptr(0x11))
	currentRootChain = chain
	BeginSJLJReplay()
	if !sjljReplaying || !Rebuilding() {
		t.Fatal("BeginSJLJReplay did not mark the replay active")
	}
	if currentRootChain != chain {
		t.Fatal("BeginSJLJReplay changed the destination root chain")
	}

	FinishSJLJReplay()
	if sjljReplaying || Rebuilding() {
		t.Fatal("FinishSJLJReplay left the replay active")
	}
}

func TestFinishSJLJReplayDoesNotEndAsyncifyRebuild(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	rebuilding = true
	FinishSJLJReplay()
	if !Rebuilding() {
		t.Fatal("SJLJ completion ended an unrelated Asyncify rebuild")
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
	active = nil
	rebuilding = false
	sjljReplaying = false
	currentRootChain = nil
}
