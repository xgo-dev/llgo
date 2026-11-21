package gcroot

import (
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
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

func TestVisitCompactFrameMaps(t *testing.T) {
	direct := new(byte)
	narrow := uint32(0x12345678)
	wide := uint64(0x12345678abcdef01)

	legacy := testStackEntry{
		stackEntry: stackEntry{m: &frameMap{numRoots: 1}},
		roots:      [2]unsafe.Pointer{unsafe.Pointer(direct)},
	}
	width32 := testStackEntry{
		stackEntry: stackEntry{
			next: &legacy.stackEntry,
			m:    &frameMap{numRoots: 2, numMeta: abi.CompactGCRootMap | 1},
		},
		roots: [2]unsafe.Pointer{unsafe.Pointer(direct), unsafe.Pointer(&narrow)},
	}
	width64 := testStackEntry{
		stackEntry: stackEntry{
			next: &width32.stackEntry,
			m:    &frameMap{numRoots: 1, numMeta: abi.CompactGCRootMap | abi.WideGCRootMap},
		},
		roots: [2]unsafe.Pointer{unsafe.Pointer(&wide)},
	}

	want := []uintptr{uintptr(wide), uintptr(unsafe.Pointer(direct)), uintptr(narrow), uintptr(unsafe.Pointer(direct))}
	var got []uintptr
	visitChain(unsafe.Pointer(&width64.stackEntry), func(root *unsafe.Pointer, meta unsafe.Pointer) {
		got = append(got, RootValue(root, meta))
	})
	if len(got) != len(want) {
		t.Fatalf("visited %d roots, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("root %d = %#x, want %#x", i, got[i], want[i])
		}
	}
}

func TestRejectsInvalidFrameMap(t *testing.T) {
	for _, m := range []*frameMap{
		nil,
		{numRoots: 1, numMeta: 2},
		{numRoots: 1, numMeta: abi.CompactGCRootMap | 1},
	} {
		entry := stackEntry{m: m}
		assertPanics(t, func() {
			visitChain(unsafe.Pointer(&entry), func(*unsafe.Pointer, unsafe.Pointer) {})
		})
	}
}

func TestRestoreChain(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	first := unsafe.Pointer(uintptr(0x11))
	second := unsafe.Pointer(uintptr(0x22))
	currentRootChain = uintptr(first)
	if got := CurrentChain(); got != first {
		t.Fatalf("current root chain = %p, want %p", got, first)
	}
	RestoreChain(second)
	if got := CurrentChain(); got != second {
		t.Fatalf("current root chain after restore = %p, want %p", got, second)
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

func TestBeginAndFinishRebuild(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	firstChain := unsafe.Pointer(uintptr(0x11))
	staleSecondChain := unsafe.Pointer(uintptr(0x22))
	var first, second Context
	currentRootChain = uintptr(firstChain)
	RegisterActive(&first)
	Register(&second)
	second.chain = staleSecondChain

	BeginRebuild(&second)
	if first.chain != firstChain {
		t.Fatalf("source chain = %p, want %p", first.chain, firstChain)
	}
	if activeContext != uintptr(unsafe.Pointer(&second)) || second.chain != nil || currentRootChain != 0 {
		t.Fatal("BeginRebuild did not switch ownership and clear stale root links")
	}
	if !Rebuilding() {
		t.Fatal("BeginRebuild did not mark the replay active")
	}

	rebuiltSecondChain := unsafe.Pointer(uintptr(0x33))
	currentRootChain = uintptr(rebuiltSecondChain)
	FinishRebuild()
	if Rebuilding() {
		t.Fatal("FinishRebuild left the replay active")
	}

	SwitchAtBoundary(&first)
	if second.chain != rebuiltSecondChain || currentRootChain != uintptr(firstChain) {
		t.Fatal("rebuilt root links were not preserved by the next switch")
	}
}

func TestBeginAndFinishSJLJReplay(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	chain := unsafe.Pointer(uintptr(0x11))
	currentRootChain = uintptr(chain)
	BeginSJLJReplay()
	if !sjljReplaying || !Rebuilding() {
		t.Fatal("BeginSJLJReplay did not mark the replay active")
	}
	if currentRootChain != uintptr(chain) {
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

func TestSuspendCurrentDropsDiscardedStack(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	var ctx Context
	RegisterActive(&ctx)
	currentRootChain = uintptr(0x55)
	SuspendCurrent(uintptr(unsafe.Pointer(&ctx)))
	if ctx.chain != nil || activeContext != 0 || currentRootChain != 0 {
		t.Fatalf("suspended context retained chain=%p active=%#x current=%#x", ctx.chain, activeContext, currentRootChain)
	}
	assertPanics(t, func() { SuspendCurrent(uintptr(unsafe.Pointer(&ctx))) })
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
	activeContext = 0
	rebuilding = false
	sjljReplaying = false
	currentRootChain = 0
}
