//go:build wasm && llgo.wasm.gc.linear

package tinygogc

import "unsafe"

const (
	finalizerActive uint8 = iota
	finalizerQueued
	finalizerRunning
	finalizerCanceled
	finalizerDone
)

type finalizerKind uint8

const (
	objectFinalizer finalizerKind = iota
	objectCleanup
)

// finalizerRecord deliberately stores an encoded object address while it is
// registered. This collector scans conservatively, so retaining an ordinary
// uintptr containing the address would keep the object alive forever.
type finalizerRecord struct {
	objectKey uintptr
	object    uintptr
	ready     unsafe.Pointer
	callback  func(unsafe.Pointer)
	state     uint8
	kind      finalizerKind
	candidate bool
	blocked   bool
	next      *finalizerRecord
	readyNext *finalizerRecord
}

var (
	finalizers              *finalizerRecord
	readyFinalizers         *finalizerRecord
	finalizerWorkerRunning  bool
	finalizerDependencyScan bool
)

// AddFinalizer registers callback for ptr without retaining the object. The
// returned function cancels a callback that has not started. Multiple callbacks
// may be registered for one object.
func AddFinalizer(ptr unsafe.Pointer, callback func(unsafe.Pointer)) (cancel func(), registered bool) {
	return addFinalizer(ptr, callback, objectFinalizer)
}

// AddCleanup registers a cleanup callback. Unlike finalizers, cleanups do not
// impose dependency ordering on one another. When an object has both, its
// cleanup remains pending until its finalizer has run and the object becomes
// unreachable again.
func AddCleanup(ptr unsafe.Pointer, callback func(unsafe.Pointer)) (cancel func(), registered bool) {
	return addFinalizer(ptr, callback, objectCleanup)
}

func addFinalizer(ptr unsafe.Pointer, callback func(unsafe.Pointer), kind finalizerKind) (cancel func(), registered bool) {
	if ptr == nil || callback == nil {
		return func() {}, false
	}

	record := &finalizerRecord{callback: callback, kind: kind}
	lock(&gcMutex)
	lazyInit()
	address := uintptr(ptr)
	if !isOnHeap(address) {
		unlock(&gcMutex)
		return func() {}, false
	}
	block := blockFromAddr(address)
	if gcStateOf(block) == blockStateFree {
		unlock(&gcMutex)
		return func() {}, false
	}
	record.objectKey = encodeFinalizerAddress(gcAddressOf(gcFindHead(block)))
	// Store the original (possibly interior) pointer in encoded form until the
	// collector has established that the object is unreachable.
	record.object = encodeFinalizerAddress(address)
	record.next = finalizers
	finalizers = record
	unlock(&gcMutex)

	return func() {
		lock(&gcMutex)
		switch record.state {
		case finalizerActive:
			unlinkFinalizer(record)
			record.state = finalizerCanceled
			record.callback = nil
		case finalizerQueued:
			record.state = finalizerCanceled
			record.ready = nil
			record.callback = nil
		}
		unlock(&gcMutex)
	}, true
}

func encodeFinalizerAddress(address uintptr) uintptr {
	return ^address
}

func unlinkFinalizer(record *finalizerRecord) {
	link := &finalizers
	for *link != nil {
		if *link == record {
			*link = record.next
			record.next = nil
			return
		}
		link = &(*link).next
	}
}

// preserveFinalizableObjects runs after ordinary roots have been marked and
// before sweeping. It moves callbacks for newly unreachable objects to the
// ready queue and marks those objects for one more cycle, so a finalizer may
// safely inspect or resurrect its argument.
//
// A finalizer on A must run before a finalizer on B when A can reach B. Extend
// the ordinary mark graph from every finalizable object once, recording each
// candidate reached through an object edge as blocked. Accumulating those
// marks discovers transitive dependencies and cycles without re-marking the
// complete live heap once per finalizer.
func preserveFinalizableObjects() {
	for record := finalizers; record != nil; record = record.next {
		record.candidate = record.state == finalizerActive && finalizerObjectState(record) == blockStateHead
		record.blocked = false
	}

	finalizerDependencyScan = true
	for record := finalizers; record != nil; record = record.next {
		if !record.candidate || record.kind != objectFinalizer || earlierFinalizerForObject(record) {
			continue
		}
		block := finalizerObjectBlock(record)
		if gcStateOf(block) == blockStateHead {
			startMark(block)
		}
	}
	finishMark()
	finalizerDependencyScan = false

	for block := uintptr(0); block < endBlock; block++ {
		state := gcStateOf(block)
		if state != blockStateHead && state != blockStateMark {
			continue
		}
		key := encodeFinalizerAddress(gcAddressOf(block))
		record := candidateForObject(key)
		if record == nil {
			continue
		}
		if hasCandidateFinalizer(key) {
			if finalizerObjectBlocked(key) || !queueCallbacksForObject(key, objectFinalizer) {
				continue
			}
		} else {
			if finalizerObjectBlocked(key) || !queueCallbacksForObject(key, objectCleanup) {
				continue
			}
		}
		if state == blockStateHead {
			startMark(block)
		}
	}

	// A cycle of finalizable objects has no valid dependency order. Preserve it
	// instead of freeing the objects while leaving callbacks registered against
	// their now-stale addresses.
	for record := finalizers; record != nil; record = record.next {
		if record.candidate && finalizerObjectState(record) == blockStateHead {
			startMark(finalizerObjectBlock(record))
		}
		record.candidate = false
		record.blocked = false
	}
	finishMark()
}

// noteFinalizerReference is called for heap edges encountered by the marker.
// During dependency discovery, reaching any candidate (including through a
// cycle back to the starting object) means that candidate cannot be finalized
// in this collection.
func noteFinalizerReference(block uintptr) {
	if !finalizerDependencyScan {
		return
	}
	key := encodeFinalizerAddress(gcAddressOf(block))
	if candidateForObject(key) != nil {
		markFinalizerObjectBlocked(key)
	}
}

func finalizerObjectBlock(record *finalizerRecord) uintptr {
	return blockFromAddr(^record.objectKey)
}

func finalizerObjectState(record *finalizerRecord) uint8 {
	return gcStateOf(finalizerObjectBlock(record))
}

// These lookups intentionally scan the callback list while the allocator is
// stopped. Building an index here would itself allocate; registered lifecycle
// callbacks are expected to remain a small set.
func candidateForObject(key uintptr) *finalizerRecord {
	for record := finalizers; record != nil; record = record.next {
		if record.candidate && record.objectKey == key {
			return record
		}
	}
	return nil
}

func earlierFinalizerForObject(record *finalizerRecord) bool {
	for candidate := finalizers; candidate != record; candidate = candidate.next {
		if candidate.candidate && candidate.kind == objectFinalizer && candidate.objectKey == record.objectKey {
			return true
		}
	}
	return false
}

func hasCandidateFinalizer(key uintptr) bool {
	for record := finalizers; record != nil; record = record.next {
		if record.candidate && record.kind == objectFinalizer && record.objectKey == key {
			return true
		}
	}
	return false
}

func finalizerObjectBlocked(key uintptr) bool {
	for record := finalizers; record != nil; record = record.next {
		if record.candidate && record.objectKey == key {
			return record.blocked
		}
	}
	return false
}

func markFinalizerObjectBlocked(key uintptr) {
	for record := finalizers; record != nil; record = record.next {
		if record.objectKey == key {
			record.blocked = true
		}
	}
}

func queueCallbacksForObject(key uintptr, kind finalizerKind) bool {
	queued := false
	link := &finalizers
	for *link != nil {
		record := *link
		if record.objectKey != key || record.kind != kind || record.state != finalizerActive || !record.candidate || record.blocked {
			link = &record.next
			continue
		}
		*link = record.next
		record.next = nil
		record.state = finalizerQueued
		original := ^record.object
		record.ready = unsafe.Pointer(original)
		record.readyNext = readyFinalizers
		readyFinalizers = record
		queued = true
	}
	return queued
}

// scheduleFinalizers starts the one serial finalizer worker when callbacks are
// ready. It must not invoke user code on the allocator's caller: that caller
// may still hold an unrelated runtime lock across an allocation. A ready
// record contains a real pointer and is reachable from a global until claimed,
// keeping the preserved object alive if another collection starts.
func scheduleFinalizers() {
	lock(&gcMutex)
	if finalizerWorkerRunning || readyFinalizers == nil {
		unlock(&gcMutex)
		return
	}
	finalizerWorkerRunning = true
	unlock(&gcMutex)
	go drainFinalizers()
}

func drainFinalizers() {
	for {
		lock(&gcMutex)
		record := readyFinalizers
		if record == nil {
			finalizerWorkerRunning = false
			unlock(&gcMutex)
			return
		}
		readyFinalizers = record.readyNext
		record.readyNext = nil
		if record.state != finalizerQueued {
			unlock(&gcMutex)
			continue
		}
		record.state = finalizerRunning
		object := record.ready
		callback := record.callback
		unlock(&gcMutex)

		callback(object)

		lock(&gcMutex)
		record.state = finalizerDone
		record.ready = nil
		record.callback = nil
		unlock(&gcMutex)
	}
}
