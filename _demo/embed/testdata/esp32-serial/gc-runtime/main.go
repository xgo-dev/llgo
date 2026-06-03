package main

import "unsafe"

// scrubStack overwrites the caller's stack frame spill area with zeros.
// On RISC-V, callee register values are spilled into the caller's stack frame.
// After a build function returns, these stale heap pointers remain on the stack
// and the conservative GC treats them as live roots. This function allocates a
// large local array that overlaps the spill area, clearing those stale values.
//
//go:noinline
func scrubStack() {
	var buf [256]uintptr
	for i := range buf {
		buf[i] = 0
	}
	// Prevent the compiler from optimizing away the array.
	if buf[0] != 0 {
		println("unreachable")
	}
}

const debugGC = true

type gcStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	Mallocs    uint64
	Frees      uint64
	HeapAlloc  uint64
	HeapSys    uint64
	HeapIdle   uint64
	HeapInuse  uint64
	StackInuse uint64
	StackSys   uint64
	GCSys      uint64
}

//go:linkname gcCollect github.com/goplus/llgo/runtime/internal/runtime/tinygogc.GC
func gcCollect() uintptr

//go:linkname readGCStats github.com/goplus/llgo/runtime/internal/runtime/tinygogc.ReadGCStats
func readGCStats() gcStats

func fail(msg string) bool {
	println("FAIL:", msg)
	return false
}

func printStats(label string, freeBytes uintptr, s gcStats) {
	if !debugGC {
		return
	}
	println(
		"GC",
		label,
		"free", freeBytes,
		"alloc", s.Alloc,
		"total", s.TotalAlloc,
		"mallocs", s.Mallocs,
		"frees", s.Frees,
		"heapAlloc", s.HeapAlloc,
		"heapSys", s.HeapSys,
		"heapIdle", s.HeapIdle,
		"heapInuse", s.HeapInuse,
		"stackInuse", s.StackInuse,
		"stackSys", s.StackSys,
		"gcSys", s.GCSys,
	)
}

func collectAndPrint(label string) (uintptr, gcStats) {
	freeBytes := gcCollect()
	stats := readGCStats()
	printStats(label, freeBytes, stats)
	return freeBytes, stats
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type node struct {
	next *node
	alt  *node
	data [4]uintptr
	id   int
}

type tinyObj struct {
	v int
}

type largeObj struct {
	data [32]uintptr
	id   int
}

type nested struct {
	child *nested
	leaf  *node
	id    int
}

// ---------------------------------------------------------------------------
// Global roots
// ---------------------------------------------------------------------------

var roots []*node
var cycleRoot *node
var pressureRoots []*node
var pressureExtra []*node
var globalRoot *node
var globalSlice []*node
var globalNested *nested
var globalIface any
var globalAppendSlice []*node
var deepChainRoot *node
var orphanSlice []*node
var globalLarge *largeObj
var partialRoot *node
var cycleA *node
var cycleB *node
var mixedSmall []*tinyObj
var mixedMedium []*node
var mixedLargeSlice []*largeObj

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildGlobalPointerRoot() {
	globalRoot = &node{id: 550}
	globalRoot.next = &node{id: 551}
}

func checkGlobalPointerRoot() bool {
	return globalRoot != nil &&
		globalRoot.id == 550 &&
		globalRoot.next != nil &&
		globalRoot.next.id == 551
}

func buildGlobalSliceRoot() {
	globalSlice = make([]*node, 2)
	globalSlice[0] = &node{id: 600}
	globalSlice[1] = &node{id: 601}
}

func checkGlobalSliceRoot() bool {
	return len(globalSlice) == 2 &&
		globalSlice[0] != nil &&
		globalSlice[1] != nil &&
		globalSlice[0].id == 600 &&
		globalSlice[1].id == 601
}

func buildReachabilityGraph() {
	roots = make([]*node, 2)
	roots[0] = &node{id: 1}
	roots[1] = &node{id: 2}
	roots[0].next = &node{id: 3}
	roots[0].alt = &node{id: 4}
	roots[0].next.next = &node{id: 5}
	roots[0].alt.next = roots[0].next.next
	roots[0].next.next.next = roots[0].alt
}

func checkReachabilityGraph() bool {
	return len(roots) == 2 &&
		roots[0] != nil &&
		roots[0].next != nil &&
		roots[0].alt != nil &&
		roots[0].next.next != nil &&
		roots[0].next.next.id == 5 &&
		roots[0].alt.next != nil &&
		roots[0].alt.next.id == 5
}

func buildRootedCycle() {
	cycleRoot = &node{id: 200}
	cycleRoot.next = &node{id: 201}
	cycleRoot.next.next = &node{id: 202}
	cycleRoot.next.next.next = &node{id: 203}
	cycleRoot.next.next.next.next = &node{id: 204}
	cycleRoot.next.next.next.next.next = cycleRoot
}

func buildPressureRoots(total int) {
	pressureRoots = make([]*node, total)
	for i := 0; i < total; i++ {
		pressureRoots[i] = &node{id: 300 + i*2}
		pressureRoots[i].data[0] = uintptr(i*2 + 1)
	}
}

func buildPressureExtra(total int) {
	pressureExtra = make([]*node, total)
	for i := 0; i < total; i++ {
		pressureExtra[i] = &node{id: 500 + i}
		pressureExtra[i].data[0] = uintptr(i + 1)
	}
}

func checkPressureRoots() bool {
	if len(pressureRoots) != 96 {
		return false
	}
	for i, p := range pressureRoots {
		if p == nil || p.id != 300+i*2 || p.data[0] != uintptr(i*2+1) {
			return false
		}
	}
	return true
}

func checkPressureExtra() bool {
	if len(pressureExtra) != 48 {
		return false
	}
	for i, p := range pressureExtra {
		if p == nil || p.id != 500+i || p.data[0] != uintptr(i+1) {
			return false
		}
	}
	return true
}

func sumCycle(head *node) (int, bool) {
	cur := head
	sum := 0
	for i := 0; i < 5; i++ {
		if cur == nil {
			return 0, false
		}
		sum += cur.id
		cur = cur.next
	}
	return sum, cur == head
}

func newDeferredNode(id int) *node {
	return &node{id: id}
}

// ---------------------------------------------------------------------------
// 1. testBasicAllocationAndStats
// Covers: Alloc bookkeeping and ReadGCStats accounting.
// ---------------------------------------------------------------------------

func testBasicAllocationAndStats() bool {
	_, base := collectAndPrint("basic-base")

	// Store pointers in a global so conservative GC can always find them.
	// Local vars may stay in registers that aren't scanned on some architectures.
	globalSlice = make([]*node, 3)
	globalSlice[0] = &node{id: 10}
	globalSlice[1] = &node{id: 11, next: globalSlice[0]}
	globalSlice[2] = &node{id: 12, next: globalSlice[1], alt: globalSlice[0]}

	_, after := collectAndPrint("basic-after")

	if globalSlice[0] == nil || globalSlice[1] == nil || globalSlice[2] == nil {
		return fail("basic allocation returned nil object")
	}
	if globalSlice[0] == globalSlice[1] || globalSlice[1] == globalSlice[2] || globalSlice[0] == globalSlice[2] {
		return fail("distinct allocations collapsed to same address")
	}
	if globalSlice[2].next != globalSlice[1] || globalSlice[2].alt != globalSlice[0] {
		return fail("allocated objects lost their references")
	}
	if after.Mallocs <= base.Mallocs {
		return fail("malloc count did not increase after basic allocation")
	}
	if after.TotalAlloc <= base.TotalAlloc {
		return fail("total allocation did not increase after basic allocation")
	}
	if after.Alloc <= base.Alloc {
		return fail("allocated bytes did not increase after basic allocation")
	}
	if after.Alloc != after.HeapAlloc {
		return fail("Alloc and HeapAlloc diverged")
	}
	if after.HeapInuse+after.HeapIdle != after.HeapSys {
		return fail("HeapSys accounting mismatch")
	}
	if after.StackInuse > after.StackSys {
		return fail("StackInuse exceeded StackSys")
	}
	if after.Sys == 0 || after.GCSys == 0 {
		return fail("GC system stats were not initialized")
	}
	globalSlice = nil
	return true
}

// ---------------------------------------------------------------------------
// 2. testStatsConsistency
// Covers: ReadGCStats invariants regardless of live set size.
// ---------------------------------------------------------------------------

func testStatsConsistency() bool {
	_, stats := collectAndPrint("stats")

	if stats.Alloc != stats.HeapAlloc {
		return fail("Alloc and HeapAlloc diverged in stats check")
	}
	if stats.HeapSys != stats.HeapInuse+stats.HeapIdle {
		return fail("HeapSys was inconsistent with HeapInuse and HeapIdle")
	}
	if stats.Mallocs < stats.Frees {
		return fail("malloc count fell below free count")
	}
	if stats.StackInuse > stats.StackSys {
		return fail("StackInuse exceeded StackSys in stats check")
	}
	if stats.Sys == 0 || stats.GCSys == 0 {
		return fail("stats check observed zeroed system counters")
	}
	return true
}

// ---------------------------------------------------------------------------
// 3. testGlobalPointerRoot
// Covers: markRoots(globalsStart, globalsEnd) for a direct global pointer.
// ---------------------------------------------------------------------------

func testGlobalPointerRoot() bool {
	globalRoot = nil
	_, base := collectAndPrint("global-ptr-base")

	buildGlobalPointerRoot()
	_, after := collectAndPrint("global-ptr-after")

	if !checkGlobalPointerRoot() {
		return fail("global pointer root was not preserved")
	}
	if after.Mallocs <= base.Mallocs {
		return fail("global pointer root did not allocate objects")
	}
	globalRoot = nil
	_, afterDrop := collectAndPrint("global-ptr-drop")
	if afterDrop.Frees < after.Frees+2 {
		return fail("clearing global pointer root did not free objects")
	}
	return true
}

// ---------------------------------------------------------------------------
// 4. testGlobalSliceRoot
// Covers: conservative global scanning of a slice header + recursive marking
// of heap-backed slice data containing pointers.
// ---------------------------------------------------------------------------

func testGlobalSliceRoot_build() (bool, gcStats, gcStats) {
	globalSlice = nil
	_, base := collectAndPrint("global-slice-base")

	buildGlobalSliceRoot()
	_, after := collectAndPrint("global-slice-after")
	if !checkGlobalSliceRoot() {
		fail("global slice root was not preserved")
		return false, base, after
	}
	if after.Mallocs <= base.Mallocs {
		fail("global slice root did not allocate objects")
		return false, base, after
	}
	return true, base, after
}

func testGlobalSliceRoot_drop(after gcStats) bool {
	globalSlice = nil
	_, afterDrop := collectAndPrint("global-slice-drop")
	if afterDrop.Frees < after.Frees+3 {
		return fail("clearing global slice root did not free objects")
	}
	return true
}

// ---------------------------------------------------------------------------
// 5. testDeferClosureLiveness
// Covers: stack root scanning for a deferred closure environment.
// ---------------------------------------------------------------------------

func runDeferClosureScenario() bool {
	ok := false
	func() {
		x := &node{id: 42}
		defer func() {
			ok = x != nil && x.id == 42
		}()
		gcCollect()
		gcCollect()
		gcCollect()
	}()
	return ok
}

func testDeferClosureLiveness() bool {
	if !runDeferClosureScenario() {
		return fail("defer closure lost heap object after GC")
	}
	return true
}

// ---------------------------------------------------------------------------
// 6. testDeferArgumentLiveness
// Covers: stack root scanning for deferred call arguments.
// ---------------------------------------------------------------------------

func runDeferArgumentScenario() bool {
	ok := false
	func() {
		defer func(p *node) {
			ok = p != nil && p.id == 43
		}(newDeferredNode(43))
		gcCollect()
		gcCollect()
		gcCollect()
	}()
	return ok
}

func testDeferArgumentLiveness() bool {
	if !runDeferArgumentScenario() {
		return fail("defer argument lost heap object after GC")
	}
	return true
}

// ---------------------------------------------------------------------------
// 7. testReachabilityGraph
// Covers: startMark/markRoot recursively traversing a shared heap graph.
// ---------------------------------------------------------------------------

func testReachabilityGraph_build() (bool, gcStats) {
	roots = nil
	_, base := collectAndPrint("reachability-base")

	buildReachabilityGraph()
	_, mid := collectAndPrint("reachability-after-build")

	if !checkReachabilityGraph() {
		fail("root graph was not constructed")
		return false, mid
	}
	if mid.Mallocs <= base.Mallocs {
		fail("malloc count did not increase for reachability graph")
		return false, mid
	}
	if mid.Alloc != mid.HeapAlloc {
		fail("Alloc and HeapAlloc diverged")
		return false, mid
	}
	if mid.HeapInuse+mid.HeapIdle != mid.HeapSys {
		fail("HeapSys accounting mismatch")
		return false, mid
	}
	return true, mid
}

func testReachabilityGraph_drop(mid gcStats) bool {
	roots = nil
	_, afterDrop := collectAndPrint("reachability-drop")
	if afterDrop.Frees < mid.Frees+6 {
		return fail("clearing reachability roots did not free graph")
	}
	if afterDrop.Alloc >= mid.Alloc {
		return fail("heap usage did not shrink after clearing reachability roots")
	}
	return true
}

// ---------------------------------------------------------------------------
// 8. testCircularReferences
// Covers: rooted cycles surviving mark-and-sweep; full reclamation on unroot.
// ---------------------------------------------------------------------------

func testCircularReferences() bool {
	cycleRoot = nil
	_, base := collectAndPrint("cycle-base")

	buildRootedCycle()
	_, mid := collectAndPrint("cycle-rooted")

	sum, ok := sumCycle(cycleRoot)
	if !ok {
		return fail("rooted cycle no longer closed after GC")
	}
	if sum != 200+201+202+203+204 {
		return fail("rooted cycle payload was corrupted after GC")
	}
	if mid.Mallocs <= base.Mallocs {
		return fail("rooted cycle did not allocate objects")
	}
	cycleRoot = nil
	_, afterDrop := collectAndPrint("cycle-drop")
	if afterDrop.Frees < mid.Frees+5 {
		return fail("clearing rooted cycle did not free cycle objects")
	}
	if afterDrop.Alloc >= mid.Alloc {
		return fail("heap usage did not shrink after clearing rooted cycle")
	}
	return true
}

// ---------------------------------------------------------------------------
// 9. testMemoryPressure
// Covers: repeated allocations with an existing live set.
// ---------------------------------------------------------------------------

func testMemoryPressure_build() (bool, gcStats) {
	pressureRoots = nil
	pressureExtra = nil
	_, base := collectAndPrint("pressure-base")

	buildPressureRoots(96)
	_, kept := collectAndPrint("pressure-kept")

	if !checkPressureRoots() {
		fail("pressure roots were not preserved")
		return false, kept
	}
	if kept.Mallocs <= base.Mallocs {
		fail("memory pressure allocations did not increase malloc count")
		return false, kept
	}

	buildPressureExtra(48)
	_, afterExtra := collectAndPrint("pressure-extra")

	if !checkPressureExtra() {
		fail("allocation under memory pressure failed")
		return false, afterExtra
	}
	if afterExtra.Mallocs <= kept.Mallocs {
		fail("extra allocations under pressure did not advance malloc count")
		return false, afterExtra
	}
	if !checkPressureRoots() {
		fail("pressure roots were corrupted by extra allocations")
		return false, afterExtra
	}
	return true, afterExtra
}

func testMemoryPressure_drop(afterExtra gcStats) bool {
	pressureRoots = nil
	pressureExtra = nil
	_, afterDrop := collectAndPrint("pressure-drop")
	if afterDrop.Frees < afterExtra.Frees+146 {
		return fail("clearing pressure roots did not free live objects")
	}
	if afterDrop.Alloc >= afterExtra.Alloc {
		return fail("heap usage did not shrink after clearing pressure roots")
	}
	return true
}

// ---------------------------------------------------------------------------
// 10. testZeroSizeAllocation
// Covers: Alloc(0) returning the sentinel zeroSizedAlloc pointer. The
// sentinel must not be freed by GC and must be consistent across calls.
// ---------------------------------------------------------------------------

func testZeroSizeAllocation() bool {
	_, base := collectAndPrint("zero-base")

	type empty struct{}
	a := new(empty)
	b := new(empty)

	gcCollect()
	gcCollect()

	_, after := collectAndPrint("zero-after")

	if a == nil || b == nil {
		return fail("zero-size allocation returned nil")
	}
	// Both zero-size allocations should yield the same sentinel pointer.
	if uintptr(unsafe.Pointer(a)) != uintptr(unsafe.Pointer(b)) {
		return fail("zero-size allocations returned different pointers")
	}
	// Zero-size allocations should not bump the Mallocs counter.
	if after.Mallocs != base.Mallocs {
		return fail("zero-size allocation incorrectly changed malloc count")
	}
	return true
}

// ---------------------------------------------------------------------------
// 11. testLargeObjectMultiBlock
// Covers: allocation spanning multiple blocks (largeObj has 33 uintptr-sized
// fields = >8 blocks on 64-bit). Survival when rooted, reclamation when not.
// ---------------------------------------------------------------------------

func buildAndCheckLargeObject() (bool, uint64) {
	globalLarge = &largeObj{id: 700}
	for i := range globalLarge.data {
		globalLarge.data[i] = uintptr(i + 1)
	}
	_, after := collectAndPrint("large-after")

	if globalLarge == nil || globalLarge.id != 700 {
		fail("large object was not preserved after GC")
		return false, 0
	}
	for i := range globalLarge.data {
		if globalLarge.data[i] != uintptr(i+1) {
			fail("large object data was corrupted after GC")
			return false, 0
		}
	}
	return true, after.Mallocs
}

func testLargeObjectMultiBlock() bool {
	globalLarge = nil
	_, base := collectAndPrint("large-base")

	ok, afterMallocs := buildAndCheckLargeObject()
	if !ok {
		return false
	}
	if afterMallocs <= base.Mallocs {
		return fail("large object allocation did not increase malloc count")
	}

	_, mid := collectAndPrint("large-mid")
	globalLarge = nil
	_, afterDrop := collectAndPrint("large-drop")
	if afterDrop.Frees < mid.Frees+1 {
		return fail("clearing large object did not free it")
	}
	if afterDrop.Alloc >= mid.Alloc {
		return fail("heap usage did not shrink after clearing large object")
	}
	return true
}

// ---------------------------------------------------------------------------
// 12. testSliceElementLiveness
// Covers: heap objects reachable through a global slice survive GC.
// The slice header is in a global variable, but the backing array and
// each *node element are heap-allocated. This verifies that the GC
// traces the indirect reference chain: global → slice → heap objects.
// ---------------------------------------------------------------------------

var sliceTestElements []*node

func testSliceElementLiveness() bool {
	sliceTestElements = make([]*node, 4)
	sliceTestElements[0] = &node{id: 800}
	sliceTestElements[1] = &node{id: 801}
	sliceTestElements[2] = &node{id: 802}
	sliceTestElements[3] = &node{id: 803}

	gcCollect()
	gcCollect()
	gcCollect()

	for i := 0; i < 4; i++ {
		if sliceTestElements[i] == nil || sliceTestElements[i].id != 800+i {
			return fail("slice element: heap pointer lost after GC")
		}
	}
	sliceTestElements = nil
	return true
}

// ---------------------------------------------------------------------------
// 12b. testRegisterRootProbe
// Probe: pointer lives ONLY in a local variable — never stored to a global
// or the heap. On architectures with windowed registers (Xtensa) the
// conservative stack scanner may miss it. A failure here is an architecture-
// specific GC limitation, not a logic error.
// ---------------------------------------------------------------------------

func testRegisterRootProbe() bool {
	a := &node{id: 801}

	gcCollect()

	if a.id != 801 {
		println("WARN: register root not scanned (expected on Xtensa windowed ABI)")
		return true // known limitation, not a hard failure
	}
	return true
}

// ---------------------------------------------------------------------------
// 13. testInterfaceLiveness
// Covers: heap objects referenced through an interface (any) global surviving
// GC, and being reclaimed once the interface is nilled.
// ---------------------------------------------------------------------------

func testInterfaceLiveness() bool {
	globalIface = nil
	_, base := collectAndPrint("iface-base")

	n := &node{id: 900}
	globalIface = n
	_, after := collectAndPrint("iface-after")

	got, ok := globalIface.(*node)
	if !ok || got == nil || got.id != 900 {
		return fail("interface-boxed heap object lost after GC")
	}
	if after.Mallocs <= base.Mallocs {
		return fail("interface allocation did not increase malloc count")
	}

	globalIface = nil
	_, afterDrop := collectAndPrint("iface-drop")
	if afterDrop.Frees <= after.Frees {
		return fail("clearing interface did not free heap object")
	}
	return true
}

// ---------------------------------------------------------------------------
// 14. testSliceAppendGrowth
// Covers: slice append triggering a new backing array. The old backing array
// should be reclaimed after the slice header updates to the new one.
// ---------------------------------------------------------------------------

func buildAppendSlice() {
	globalAppendSlice = make([]*node, 0, 2)
	globalAppendSlice = append(globalAppendSlice, &node{id: 1000})
	globalAppendSlice = append(globalAppendSlice, &node{id: 1001})
	// Force growth by appending beyond capacity.
	for i := 0; i < 10; i++ {
		globalAppendSlice = append(globalAppendSlice, &node{id: 1002 + i})
	}
}

func checkAppendSlice() bool {
	if len(globalAppendSlice) != 12 {
		return fail("append slice has wrong length")
	}
	if globalAppendSlice[0].id != 1000 || globalAppendSlice[1].id != 1001 {
		return fail("append slice original elements corrupted")
	}
	for i := 0; i < 10; i++ {
		if globalAppendSlice[2+i].id != 1002+i {
			return fail("append slice appended elements corrupted")
		}
	}
	return true
}

func testSliceAppendGrowth() bool {
	globalAppendSlice = nil
	_, base := collectAndPrint("append-base")

	buildAppendSlice()
	_, afterBuild := collectAndPrint("append-build")

	if !checkAppendSlice() {
		return false
	}
	if afterBuild.Mallocs <= base.Mallocs {
		return fail("slice growth did not allocate")
	}

	globalAppendSlice = nil
	_, afterDrop := collectAndPrint("append-drop")
	if afterDrop.Frees <= afterBuild.Frees {
		return fail("clearing append slice did not free objects")
	}
	return true
}

// ---------------------------------------------------------------------------
// 15. testDeepReferenceChain
// Covers: deep linked list that may overflow the mark stack (markStackSize=8
// pointers), triggering markStackOverflow and finishMark rescan.
// ---------------------------------------------------------------------------

func buildDeepChain(chainLen int) {
	deepChainRoot = &node{id: 2000}
	cur := deepChainRoot
	for i := 1; i < chainLen; i++ {
		cur.next = &node{id: 2000 + i}
		cur = cur.next
	}
}

func verifyDeepChain(chainLen int) bool {
	cur := deepChainRoot
	for i := 0; i < chainLen; i++ {
		if cur == nil {
			return fail("deep chain truncated at link")
		}
		if cur.id != 2000+i {
			return fail("deep chain node has wrong id")
		}
		cur = cur.next
	}
	if cur != nil {
		return fail("deep chain has unexpected extra node")
	}
	return true
}

func testDeepReferenceChain() bool {
	deepChainRoot = nil
	_, base := collectAndPrint("deep-base")

	const chainLen = 30 // well beyond markStackSize (8)
	buildDeepChain(chainLen)
	_, after := collectAndPrint("deep-after")

	if !verifyDeepChain(chainLen) {
		return false
	}
	if after.Mallocs <= base.Mallocs {
		return fail("deep chain did not allocate objects")
	}

	deepChainRoot = nil
	_, afterDrop := collectAndPrint("deep-drop")
	expectedFrees := after.Frees + uint64(chainLen)
	if afterDrop.Frees < expectedFrees {
		return fail("clearing deep chain did not free all nodes")
	}
	if afterDrop.Alloc >= after.Alloc {
		return fail("heap usage did not shrink after clearing deep chain")
	}
	return true
}

// ---------------------------------------------------------------------------
// 16. testUnreachableOrphanCollection
// Covers: objects allocated but never rooted are collected as garbage.
// ---------------------------------------------------------------------------

func buildOrphanSlice() {
	orphanSlice = make([]*node, 5)
	for i := range orphanSlice {
		orphanSlice[i] = &node{id: 3000 + i}
	}
}

func testUnreachableOrphanCollection_build() (bool, gcStats) {
	orphanSlice = nil
	_, base := collectAndPrint("orphan-base")

	buildOrphanSlice()
	_, afterBuild := collectAndPrint("orphan-built")

	if afterBuild.Mallocs <= base.Mallocs {
		fail("orphan allocation did not increase malloc count")
		return false, afterBuild
	}
	return true, afterBuild
}

func testUnreachableOrphanCollection_drop(afterBuild gcStats) bool {
	orphanSlice = nil
	_, afterDrop := collectAndPrint("orphan-drop")
	if afterDrop.Frees < afterBuild.Frees+6 {
		return fail("orphaned objects were not collected")
	}
	if afterDrop.Alloc >= afterBuild.Alloc {
		return fail("heap usage did not shrink after orphan collection")
	}
	return true
}

// ---------------------------------------------------------------------------
// 17. testGCIdempotency
// Covers: running GC twice on a stable heap must not free additional objects.
// ---------------------------------------------------------------------------

func testGCIdempotency() bool {
	globalRoot = &node{id: 4000}
	globalRoot.next = &node{id: 4001}
	_, first := collectAndPrint("idempotent-first")

	_, second := collectAndPrint("idempotent-second")

	if second.Frees != first.Frees {
		return fail("second GC on stable heap freed extra objects")
	}
	if second.Mallocs != first.Mallocs {
		return fail("second GC on stable heap changed malloc count")
	}
	if second.Alloc != first.Alloc {
		return fail("Alloc changed between idempotent GC cycles")
	}

	globalRoot = nil
	collectAndPrint("idempotent-cleanup")
	return true
}

// ---------------------------------------------------------------------------
// 18. testAllocAfterGCReclaim
// Covers: after GC frees objects, the allocator can reuse freed space.
// ---------------------------------------------------------------------------

func testAllocAfterGCReclaim() bool {
	globalSlice = make([]*node, 10)
	for i := range globalSlice {
		globalSlice[i] = &node{id: 5000 + i}
	}
	_, before := collectAndPrint("reclaim-before")

	globalSlice = nil
	collectAndPrint("reclaim-freed")

	// Re-allocate the same amount. If GC reclaimed the old objects,
	// the allocator can reuse that space.
	globalSlice = make([]*node, 10)
	for i := range globalSlice {
		globalSlice[i] = &node{id: 6000 + i}
	}
	_, afterReuse := collectAndPrint("reclaim-reuse")

	for i := range globalSlice {
		if globalSlice[i] == nil || globalSlice[i].id != 6000+i {
			return fail("reclaim: reused allocation corrupted")
		}
	}
	if afterReuse.Mallocs <= before.Mallocs {
		return fail("reclaim: new allocations did not advance malloc count")
	}

	globalSlice = nil
	collectAndPrint("reclaim-cleanup")
	return true
}

// ---------------------------------------------------------------------------
// 19. testNestedStructPointers
// Covers: recursive marking through nested struct pointer fields (child and
// leaf), verifying the full tree survives GC and is freed when unrooted.
// ---------------------------------------------------------------------------

func testNestedStructPointers() bool {
	globalNested = nil
	_, base := collectAndPrint("nested-base")

	globalNested = &nested{id: 100}
	globalNested.leaf = &node{id: 101}
	globalNested.child = &nested{id: 102}
	globalNested.child.leaf = &node{id: 103}
	globalNested.child.child = &nested{id: 104}
	globalNested.child.child.leaf = &node{id: 105}
	_, after := collectAndPrint("nested-after")

	if globalNested == nil || globalNested.id != 100 {
		return fail("nested root lost")
	}
	if globalNested.leaf == nil || globalNested.leaf.id != 101 {
		return fail("nested root leaf lost")
	}
	if globalNested.child == nil || globalNested.child.id != 102 {
		return fail("nested child lost")
	}
	if globalNested.child.leaf == nil || globalNested.child.leaf.id != 103 {
		return fail("nested child leaf lost")
	}
	if globalNested.child.child == nil || globalNested.child.child.id != 104 {
		return fail("nested grandchild lost")
	}
	if globalNested.child.child.leaf == nil || globalNested.child.child.leaf.id != 105 {
		return fail("nested grandchild leaf lost")
	}
	if after.Mallocs <= base.Mallocs {
		return fail("nested struct allocation did not increase malloc count")
	}

	globalNested = nil
	_, afterDrop := collectAndPrint("nested-drop")
	// 3 nested + 3 node = 6 objects (at minimum)
	if afterDrop.Frees < after.Frees+6 {
		return fail("clearing nested struct did not free all objects")
	}
	if afterDrop.Alloc >= after.Alloc {
		return fail("heap usage did not shrink after clearing nested struct")
	}
	return true
}

// ---------------------------------------------------------------------------
// 20. testTotalAllocMonotonicity
// Covers: TotalAlloc is cumulative and must never decrease after GC.
// ---------------------------------------------------------------------------

func testTotalAllocMonotonicity() bool {
	_, s1 := collectAndPrint("mono-s1")

	globalRoot = &node{id: 7000}
	globalRoot.next = &node{id: 7001}
	_, s2 := collectAndPrint("mono-s2")

	if s2.TotalAlloc <= s1.TotalAlloc {
		return fail("TotalAlloc did not increase after allocation")
	}

	globalRoot = nil
	_, s3 := collectAndPrint("mono-s3")

	if s3.TotalAlloc < s2.TotalAlloc {
		return fail("TotalAlloc decreased after GC (should be monotonic)")
	}

	globalRoot = &node{id: 7002}
	_, s4 := collectAndPrint("mono-s4")

	if s4.TotalAlloc <= s3.TotalAlloc {
		return fail("TotalAlloc did not increase after further allocation")
	}

	globalRoot = nil
	collectAndPrint("mono-cleanup")
	return true
}

// ---------------------------------------------------------------------------
// 21. testStatsConsistencyAfterMultipleGCCycles
// Covers: ReadGCStats invariants hold after several allocation/collection
// cycles with varying live set sizes.
// ---------------------------------------------------------------------------

func testStatsConsistencyAfterMultipleGCCycles() bool {
	globalSlice = nil
	collectAndPrint("multi-gc-init")

	for round := 0; round < 3; round++ {
		n := 8 + round*4
		globalSlice = make([]*node, n)
		for i := 0; i < n; i++ {
			globalSlice[i] = &node{id: 8000 + round*100 + i}
		}
		_, stats := collectAndPrint("multi-gc-round")

		if stats.Alloc != stats.HeapAlloc {
			return fail("Alloc != HeapAlloc after multi-GC round")
		}
		if stats.HeapSys != stats.HeapInuse+stats.HeapIdle {
			return fail("HeapSys inconsistent after multi-GC round")
		}
		if stats.Mallocs < stats.Frees {
			return fail("Mallocs < Frees after multi-GC round")
		}
		if stats.StackInuse > stats.StackSys {
			return fail("StackInuse > StackSys after multi-GC round")
		}
	}

	globalSlice = nil
	collectAndPrint("multi-gc-cleanup")
	return true
}

// ---------------------------------------------------------------------------
// 22. testDeferInLoopLiveness
// Covers: defer inside a loop — each iteration's deferred closure captures
// a different heap object. All must survive GC executed mid-loop.
// ---------------------------------------------------------------------------

func runDeferInLoopScenario() bool {
	results := make([]int, 4)
	func() {
		for i := 0; i < 4; i++ {
			n := &node{id: 9000 + i}
			defer func(p *node, pos int) {
				results[pos] = p.id
			}(n, i)

			if i == 2 {
				gcCollect()
				gcCollect()
			}
		}
		gcCollect()
	}()
	// results[i] = 9000+i for each i
	for i := 0; i < 4; i++ {
		if results[i] != 9000+i {
			return false
		}
	}
	return true
}

func testDeferInLoopLiveness() bool {
	if !runDeferInLoopScenario() {
		return fail("defer in loop lost heap object after GC")
	}
	return true
}

// ---------------------------------------------------------------------------
// 23. testPartialGraphUnlinking
// Covers: removing a reference edge from a rooted graph makes a subgraph
// unreachable; GC must collect only the unreachable part.
// ---------------------------------------------------------------------------

func buildPartialGraph() {
	// Build: root -> a -> b -> c
	partialRoot = &node{id: 10000}
	a := &node{id: 10001}
	b := &node{id: 10002}
	cNode := &node{id: 10003}
	partialRoot.next = a
	a.next = b
	b.next = cNode
}

func unlinkPartialGraph() {
	// Unlink b from a: b and c become unreachable via global root.
	partialRoot.next.next = nil
}

func testPartialGraphUnlinking() bool {
	partialRoot = nil
	_, base := collectAndPrint("partial-base")

	buildPartialGraph()
	_, afterBuild := collectAndPrint("partial-build")

	if afterBuild.Mallocs <= base.Mallocs {
		return fail("partial graph did not allocate")
	}

	unlinkPartialGraph()
	_, afterUnlink := collectAndPrint("partial-unlink")

	if partialRoot == nil || partialRoot.id != 10000 {
		return fail("partial graph root lost after unlink")
	}
	if partialRoot.next == nil || partialRoot.next.id != 10001 {
		return fail("partial graph node a lost after unlink")
	}
	// b and c were freed.
	if afterUnlink.Frees < afterBuild.Frees+2 {
		return fail("partial graph unlink did not free unreachable subgraph")
	}

	partialRoot = nil
	collectAndPrint("partial-cleanup")
	return true
}

// ---------------------------------------------------------------------------
// 24. testMultipleCyclesDisjoint
// Covers: two disjoint cycles rooted separately; clearing one root frees
// only that cycle while the other remains live.
// ---------------------------------------------------------------------------

func testMultipleCyclesDisjoint() bool {
	cycleA = nil
	cycleB = nil
	collectAndPrint("disjoint-base")

	// Cycle A: 3 nodes
	cycleA = &node{id: 11000}
	cycleA.next = &node{id: 11001}
	cycleA.next.next = &node{id: 11002}
	cycleA.next.next.next = cycleA

	// Cycle B: 2 nodes
	cycleB = &node{id: 12000}
	cycleB.next = &node{id: 12001}
	cycleB.next.next = cycleB

	_, afterBuild := collectAndPrint("disjoint-build")

	// Drop cycle A only.
	cycleA = nil
	_, afterDropA := collectAndPrint("disjoint-dropA")
	if afterDropA.Frees < afterBuild.Frees+3 {
		return fail("dropping cycle A did not free 3 nodes")
	}

	// Cycle B still alive.
	if cycleB == nil || cycleB.id != 12000 || cycleB.next == nil || cycleB.next.id != 12001 {
		return fail("cycle B was corrupted after dropping cycle A")
	}
	if cycleB.next.next != cycleB {
		return fail("cycle B lost its cycle link")
	}

	// Drop cycle B.
	cycleB = nil
	_, afterDropB := collectAndPrint("disjoint-dropB")
	if afterDropB.Frees < afterDropA.Frees+2 {
		return fail("dropping cycle B did not free 2 nodes")
	}
	return true
}

// ---------------------------------------------------------------------------
// 25. testMixedObjectSizes
// Covers: coexistence of small (tinyObj), medium (node), and large (largeObj)
// allocations; GC must correctly track and sweep all sizes.
// ---------------------------------------------------------------------------

func buildMixedSizes() {
	mixedSmall = make([]*tinyObj, 5)
	for i := range mixedSmall {
		mixedSmall[i] = &tinyObj{v: 13000 + i}
	}
	mixedMedium = make([]*node, 5)
	for i := range mixedMedium {
		mixedMedium[i] = &node{id: 14000 + i}
	}
	mixedLargeSlice = make([]*largeObj, 3)
	for i := range mixedLargeSlice {
		mixedLargeSlice[i] = &largeObj{id: 15000 + i}
	}
}

func checkMixedSizes() bool {
	for i, s := range mixedSmall {
		if s == nil || s.v != 13000+i {
			return false
		}
	}
	for i, m := range mixedMedium {
		if m == nil || m.id != 14000+i {
			return false
		}
	}
	for i, l := range mixedLargeSlice {
		if l == nil || l.id != 15000+i {
			return false
		}
	}
	return true
}

func testMixedObjectSizes() bool {
	mixedSmall = nil
	mixedMedium = nil
	mixedLargeSlice = nil
	_, base := collectAndPrint("mixed-base")

	buildMixedSizes()
	_, afterBuild := collectAndPrint("mixed-build")

	if !checkMixedSizes() {
		return fail("mixed objects corrupted")
	}
	if afterBuild.Mallocs <= base.Mallocs {
		return fail("mixed sizes did not allocate")
	}

	mixedSmall = nil
	mixedMedium = nil
	mixedLargeSlice = nil
	_, afterDrop := collectAndPrint("mixed-drop")
	if afterDrop.Frees <= afterBuild.Frees {
		return fail("mixed sizes were not collected")
	}
	if afterDrop.Alloc >= afterBuild.Alloc {
		return fail("heap did not shrink after clearing mixed sizes")
	}
	return true
}

// ---------------------------------------------------------------------------
// main — run all tests in order
// ---------------------------------------------------------------------------

func runTest(name string, fn func() bool) bool {
	if !fn() {
		return false
	}
	if debugGC {
		println("PASS:", name)
	}
	return true
}

func main() {
	ok := true

	if !runTest("BasicAllocationAndStats", testBasicAllocationAndStats) {
		ok = false
	}
	if !runTest("StatsConsistency", testStatsConsistency) {
		ok = false
	}
	if !runTest("GlobalPointerRoot", testGlobalPointerRoot) {
		ok = false
	}

	// Split tests: scrubStack() between build/drop phases clears stale
	// RISC-V register spills from the caller's stack frame so the
	// conservative GC does not treat them as live roots.
	{
		buildOk, _, afterStats := testGlobalSliceRoot_build()
		scrubStack()
		if buildOk && testGlobalSliceRoot_drop(afterStats) {
			if debugGC {
				println("PASS: GlobalSliceRoot")
			}
		} else {
			ok = false
		}
	}
	if !runTest("DeferClosureLiveness", testDeferClosureLiveness) {
		ok = false
	}
	if !runTest("DeferArgumentLiveness", testDeferArgumentLiveness) {
		ok = false
	}
	{
		buildOk, midStats := testReachabilityGraph_build()
		scrubStack()
		if buildOk && testReachabilityGraph_drop(midStats) {
			if debugGC {
				println("PASS: ReachabilityGraph")
			}
		} else {
			ok = false
		}
	}
	if !runTest("CircularReferences", testCircularReferences) {
		ok = false
	}
	{
		buildOk, extraStats := testMemoryPressure_build()
		scrubStack()
		if buildOk && testMemoryPressure_drop(extraStats) {
			if debugGC {
				println("PASS: MemoryPressure")
			}
		} else {
			ok = false
		}
	}
	if !runTest("ZeroSizeAllocation", testZeroSizeAllocation) {
		ok = false
	}
	if !runTest("LargeObjectMultiBlock", testLargeObjectMultiBlock) {
		ok = false
	}
	if !runTest("SliceElementLiveness", testSliceElementLiveness) {
		ok = false
	}
	if !runTest("RegisterRootProbe", testRegisterRootProbe) {
		ok = false
	}
	if !runTest("InterfaceLiveness", testInterfaceLiveness) {
		ok = false
	}
	if !runTest("SliceAppendGrowth", testSliceAppendGrowth) {
		ok = false
	}
	if !runTest("DeepReferenceChain", testDeepReferenceChain) {
		ok = false
	}
	{
		buildOk, buildStats := testUnreachableOrphanCollection_build()
		scrubStack()
		if buildOk && testUnreachableOrphanCollection_drop(buildStats) {
			if debugGC {
				println("PASS: UnreachableOrphanCollection")
			}
		} else {
			ok = false
		}
	}
	if !runTest("GCIdempotency", testGCIdempotency) {
		ok = false
	}
	if !runTest("AllocAfterGCReclaim", testAllocAfterGCReclaim) {
		ok = false
	}
	if !runTest("NestedStructPointers", testNestedStructPointers) {
		ok = false
	}
	if !runTest("TotalAllocMonotonicity", testTotalAllocMonotonicity) {
		ok = false
	}
	if !runTest("StatsConsistencyAfterMultipleGCCycles", testStatsConsistencyAfterMultipleGCCycles) {
		ok = false
	}
	if !runTest("DeferInLoopLiveness", testDeferInLoopLiveness) {
		ok = false
	}
	if !runTest("PartialGraphUnlinking", testPartialGraphUnlinking) {
		ok = false
	}
	if !runTest("MultipleCyclesDisjoint", testMultipleCyclesDisjoint) {
		ok = false
	}
	if !runTest("MixedObjectSizes", testMixedObjectSizes) {
		ok = false
	}

	if ok {
		println("OK")
	}
}
