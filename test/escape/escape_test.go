package escape

import (
	"runtime"
	"sync/atomic"
	"testing"
)

type object struct {
	value   int64
	padding [7]uint64
}

var garbage *object

//go:noinline
func newObject(n int64) *object {
	p := &object{value: n}
	runtime.SetFinalizer(p, func(p *object) { atomic.StoreInt64(&p.value, -1) })
	return p
}

//go:noinline
func collect() {
	for range 3 {
		runtime.GC()
		// Reuse the payload's allocation size to expose premature reclamation.
		for range 128 {
			garbage = &object{value: -2}
		}
	}
}

//go:noinline
func invokeBox(v any) int64 {
	collect()
	return v.(func() int64)()
}

//go:noinline
func boxed(f func() int64) int64 { return invokeBox(f) }

//go:noinline
func borrowed(p *struct{ child *object }) int64 {
	collect()
	return atomic.LoadInt64(&p.child.value)
}

func TestBorrowedFieldsAndInterfaceClosureSurviveGC(t *testing.T) {
	for i := int64(1); i <= 4; i++ {
		holder := struct{ child *object }{newObject(i)}
		if got := borrowed(&holder); got != i {
			t.Fatalf("borrowed child = %d, want %d", got, i)
		}
		p := newObject(i + 10)
		if got := boxed(func() int64 { return atomic.LoadInt64(&p.value) }); got != i+10 {
			t.Fatalf("boxed closure child = %d, want %d", got, i+10)
		}
	}
}

//go:noinline
func retain(p *object) *object { return p }

func TestEscapingAliasesAndIterationIdentity(t *testing.T) {
	var all []*object
	for i := int64(0); i < 16; i++ {
		current := object{value: i}
		all = append(all, retain(&current))
	}
	collect()
	for i, p := range all {
		if p.value != int64(i) || i > 0 && p == all[i-1] {
			t.Fatalf("iteration %d reused retained storage", i)
		}
	}
}

var published atomic.Pointer[object]

//go:noinline
func publishLocal(n int64) {
	local := object{value: n}
	published.Store(&local)
}

func TestAtomicPublicationAndGoroutineCapture(t *testing.T) {
	publishLocal(23)
	collect()
	if got := published.Load().value; got != 23 {
		t.Fatalf("published object = %d", got)
	}
	published.Store(nil)
	ready, release, done := make(chan bool), make(chan bool), make(chan int64)
	func() {
		p := newObject(29)
		go func() {
			ready <- true
			<-release
			done <- atomic.LoadInt64(&p.value)
		}()
	}()
	<-ready
	collect()
	release <- true
	if got := <-done; got != 29 {
		t.Fatalf("goroutine capture = %d", got)
	}
}

//go:noinline
func deferredGC(p *object, panicNow bool) (result int64) {
	defer func() {
		collect()
		result += atomic.LoadInt64(&p.value)
		if panicNow && recover() != "escape" {
			panic("lost recovery payload")
		}
	}()
	result = 100
	if panicNow {
		panic("escape")
	}
	return
}

func TestDeferredCaptureSurvivesCollectionAndRecovery(t *testing.T) {
	for _, panicNow := range []bool{false, true} {
		if got := deferredGC(newObject(7), panicNow); got != 107 {
			t.Fatalf("panic=%v result=%d, want 107", panicNow, got)
		}
	}
}

func TestMixedDeferStorageAndForeignOwner(t *testing.T) {
	var got []int
	func() {
		defer func(n int) { got = append(got, n) }(0)
		for i := 1; i <= 3; i++ {
			defer func(n int) { collect(); got = append(got, n) }(i)
		}
		if len(got) == 0 {
			defer func(n int) { got = append(got, n) }(4)
		}
		iter := func(yield func(int) bool) {
			for i := 5; i <= 7; i++ {
				if !yield(i) {
					return
				}
			}
		}
		for i := range iter {
			defer func(n int) { collect(); got = append(got, n) }(i)
		}
	}()
	if len(got) != 8 {
		t.Fatalf("deferred calls = %v", got)
	}
	for i, value := range got {
		if value != 7-i {
			t.Fatalf("defer order = %v", got)
		}
	}
}
