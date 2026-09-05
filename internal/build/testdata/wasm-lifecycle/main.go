package main

import (
	"runtime"
	"time"
	"weak"
)

type object struct {
	value int
	next  *object
	pad   [64]byte
}

func (o *object) Value() int { return o.value }

type valued interface {
	Value() int
}

var weakObject weak.Pointer[object]
var nonCapturingEvents chan<- int

func main() {
	testFinalizerCalls()
	testFinalizerCancellation()
	testFinalizerReplacement()
	testFinalizerDependencyOrder()
	testFinalizerBeforeCleanup()
	testCleanupAndStop()
	testWeakPointer()
	testFinalizerCycle()
	println("wasm lifecycle ok")
}

//go:noinline
func installFinalizerCalls(events chan<- int) {
	installPointerFinalizer(events)
	installIntegerFinalizer(events)
	installFloatFinalizer(events)
	installAggregateFinalizer(events)
	installMultipleFinalizer(events)
	installInterfaceFinalizer(events)
	installInterfaceAggregateFinalizer(events)
	installFunctionFinalizer(events)
	installComplexFinalizer(events)
	installZeroResultFinalizer(events)
	installNonCapturingFinalizers(events)
}

// Keep each target in a separate frame. The linear collector is conservative,
// so unrelated stale pointer-shaped words must not make this ABI test flaky.
//
//go:noinline
func installPointerFinalizer(events chan<- int) {
	captured := 1
	pointer := &object{value: 40}
	runtime.SetFinalizer(pointer, func(value *object) int {
		if value.value+captured != 41 {
			panic("capturing pointer finalizer received the wrong object")
		}
		events <- 1
		return value.value
	})
}

//go:noinline
func installIntegerFinalizer(events chan<- int) {
	integer := &object{value: 42}
	runtime.SetFinalizer(integer, func(value *object) int64 {
		events <- 2
		return int64(value.value)
	})
}

//go:noinline
func installFloatFinalizer(events chan<- int) {
	floating := &object{value: 43}
	runtime.SetFinalizer(floating, func(value *object) float64 {
		events <- 3
		return float64(value.value)
	})
}

//go:noinline
func installAggregateFinalizer(events chan<- int) {
	aggregate := &object{value: 44}
	runtime.SetFinalizer(aggregate, func(value *object) struct{ first, second int } {
		events <- 4
		return struct{ first, second int }{value.value, value.value + 1}
	})
}

//go:noinline
func installMultipleFinalizer(events chan<- int) {
	multiple := &object{value: 45}
	runtime.SetFinalizer(multiple, func(value *object) (int, float64) {
		events <- 5
		return value.value, float64(value.value)
	})
}

//go:noinline
func installInterfaceFinalizer(events chan<- int) {
	iface := &object{value: 46}
	runtime.SetFinalizer(iface, func(value valued) float32 {
		if value.Value() != 46 {
			panic("interface finalizer received the wrong object")
		}
		events <- 6
		return float32(value.Value())
	})
}

//go:noinline
func installInterfaceAggregateFinalizer(events chan<- int) {
	iface := &object{value: 47}
	runtime.SetFinalizer(iface, func(value valued) struct{ first, second int } {
		if value.Value() != 47 {
			panic("aggregate interface finalizer received the wrong object")
		}
		events <- 7
		return struct{ first, second int }{value.Value(), value.Value() + 1}
	})
}

//go:noinline
func installFunctionFinalizer(events chan<- int) {
	value := &object{value: 48}
	runtime.SetFinalizer(value, func(*object) func() {
		events <- 8
		return func() {}
	})
}

//go:noinline
func installComplexFinalizer(events chan<- int) {
	value := &object{value: 49}
	runtime.SetFinalizer(value, func(*object) complex64 {
		events <- 9
		return 1 + 2i
	})
}

//go:noinline
func installZeroResultFinalizer(events chan<- int) {
	value := &object{value: 50}
	runtime.SetFinalizer(value, func(*object) (struct{}, [0]byte) {
		events <- 10
		return struct{}{}, [0]byte{}
	})
}

type aggregateResult struct {
	first  int
	second int
}

func nonCapturingPointerFinalizer(value *object) aggregateResult {
	nonCapturingEvents <- 11
	return aggregateResult{value.value, value.value + 1}
}

func nonCapturingInterfaceFinalizer(value valued) aggregateResult {
	nonCapturingEvents <- 12
	return aggregateResult{value.Value(), value.Value() + 1}
}

//go:noinline
func installNonCapturingFinalizers(events chan<- int) {
	nonCapturingEvents = events
	pointer := &object{value: 51}
	iface := &object{value: 52}
	runtime.SetFinalizer(pointer, nonCapturingPointerFinalizer)
	runtime.SetFinalizer(iface, nonCapturingInterfaceFinalizer)
}

func testFinalizerCalls() {
	events := make(chan int, 12)
	done := make(chan struct{})
	go func() {
		installFinalizerCalls(events)
		close(done)
	}()
	<-done
	collectUntil("typed finalizers", func() bool { return len(events) == 12 })

	seen := [13]bool{}
	for range 12 {
		event := <-events
		if event < 1 || event > 12 || seen[event] {
			panic("typed finalizer ran with an invalid or duplicate event")
		}
		seen[event] = true
	}
	nonCapturingEvents = nil
}

//go:noinline
func installCanceledFinalizer(events chan<- int) {
	value := &object{value: 47}
	runtime.SetFinalizer(value, func(*object) { events <- 1 })
	runtime.SetFinalizer(value, nil)
}

func testFinalizerCancellation() {
	events := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		installCanceledFinalizer(events)
		close(done)
	}()
	<-done
	collectCycles(4)
	if len(events) != 0 {
		panic("canceled finalizer ran")
	}
}

//go:noinline
func installReplacementFinalizer(events chan<- int) {
	value := &object{value: 51}
	runtime.SetFinalizer(value, func(*object) { events <- 1 })
	runtime.SetFinalizer(value, func(*object) { events <- 2 })
}

func testFinalizerReplacement() {
	events := make(chan int, 2)
	done := make(chan struct{})
	go func() {
		installReplacementFinalizer(events)
		close(done)
	}()
	<-done
	collectUntil("replacement finalizer", func() bool { return len(events) != 0 })
	if event := <-events; event != 2 {
		panic("replaced finalizer ran")
	}
	collectCycles(4)
	if len(events) != 0 {
		panic("replaced finalizer ran after replacement")
	}
}

//go:noinline
func installDependentFinalizers(events chan<- int) {
	// Allocate the dependency first, so an address-ordered collector encounters
	// it before the object whose finalizer must run first.
	dependency := &object{value: 2}
	owner := &object{value: 1, next: dependency}
	runtime.SetFinalizer(dependency, func(*object) { events <- 2 })
	runtime.SetFinalizer(owner, func(value *object) {
		if value.next == nil || value.next.value != 2 {
			panic("finalizer dependency was not preserved")
		}
		events <- 1
	})
}

func testFinalizerDependencyOrder() {
	events := make(chan int, 2)
	done := make(chan struct{})
	go func() {
		installDependentFinalizers(events)
		close(done)
	}()
	<-done
	collectUntil("owner finalizer", func() bool { return len(events) != 0 })
	if event := <-events; event != 1 {
		panic("dependency finalizer ran before owner finalizer")
	}
	if len(events) != 0 {
		panic("dependent finalizers ran in the same collection")
	}
	collectUntil("dependency finalizer", func() bool { return len(events) != 0 })
	if event := <-events; event != 2 {
		panic("dependency finalizer produced the wrong event")
	}
}

//go:noinline
func installFinalizerCycle(events chan<- int) {
	first := &object{value: 1}
	second := &object{value: 2}
	first.next = second
	second.next = first
	runtime.SetFinalizer(first, func(*object) { events <- 1 })
	runtime.SetFinalizer(second, func(*object) { events <- 2 })
}

func testFinalizerCycle() {
	events := make(chan int, 2)
	done := make(chan struct{})
	go func() {
		installFinalizerCycle(events)
		close(done)
	}()
	<-done
	collectCycles(4)
	if len(events) != 0 {
		panic("cyclic finalizers ran without a valid dependency order")
	}
}

//go:noinline
func installFinalizerAndCleanup(events chan<- int) {
	value := &object{value: 48}
	runtime.SetFinalizer(value, func(*object) { events <- 1 })
	runtime.AddCleanup(value, func(int) { events <- 2 }, 48)
}

func testFinalizerBeforeCleanup() {
	events := make(chan int, 2)
	done := make(chan struct{})
	go func() {
		installFinalizerAndCleanup(events)
		close(done)
	}()
	<-done
	collectUntil("object finalizer", func() bool { return len(events) != 0 })
	if event := <-events; event != 1 {
		panic("cleanup ran before object finalizer")
	}
	if len(events) != 0 {
		panic("cleanup ran in the finalizer collection")
	}
	collectUntil("post-finalizer cleanup", func() bool { return len(events) != 0 })
	if event := <-events; event != 2 {
		panic("post-finalizer cleanup produced the wrong event")
	}
}

//go:noinline
func installCleanups(events chan<- int) {
	active := &object{value: 49}
	runtime.AddCleanup(active, func(value int) {
		if value != 49 {
			panic("cleanup received the wrong argument")
		}
		// Call back into the collector to verify lifecycle callbacks run on the
		// finalizer worker, outside the allocator lock.
		runtime.GC()
		events <- 1
	}, 49)

	stopped := &object{value: 50}
	cleanup := runtime.AddCleanup(stopped, func(int) { events <- 2 }, 50)
	cleanup.Stop()
	cleanup.Stop()
	runtime.KeepAlive(active)
	runtime.KeepAlive(stopped)
}

func testCleanupAndStop() {
	events := make(chan int, 2)
	done := make(chan struct{})
	go func() {
		installCleanups(events)
		close(done)
	}()
	<-done
	collectUntil("active cleanup", func() bool { return len(events) != 0 })
	if event := <-events; event != 1 {
		panic("stopped cleanup ran")
	}
	collectCycles(4)
	if len(events) != 0 {
		panic("stopped cleanup ran after repeated collection")
	}
}

//go:noinline
func installWeakPointer() {
	value := &object{value: 51}
	weakObject = weak.Make(value)
	if got := weakObject.Value(); got == nil || got.value != 51 {
		panic("weak pointer did not preserve a live object")
	}
}

//go:noinline
func weakObjectAlive() bool {
	return weakObject.Value() != nil
}

func testWeakPointer() {
	done := make(chan struct{})
	go func() {
		installWeakPointer()
		close(done)
	}()
	<-done
	collectUntil("weak pointer expiration", func() bool { return !weakObjectAlive() })
}

func collectUntil(name string, done func() bool) {
	for range 24 {
		clobberStack(16, 1)
		runtime.GC()
		time.Sleep(time.Millisecond)
		if done() {
			return
		}
	}
	panic(name + " did not complete")
}

func collectCycles(count int) {
	for range count {
		clobberStack(16, 1)
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
}

//go:noinline
func clobberStack(depth int, value uintptr) uintptr {
	var words [32]uintptr
	for i := range words {
		words[i] = value + uintptr(i)
	}
	if depth != 0 {
		return words[depth%len(words)] + clobberStack(depth-1, value+1)
	}
	return words[0]
}
