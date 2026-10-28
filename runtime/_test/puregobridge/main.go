//go:build linux && amd64

package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	bridge "github.com/xgo-dev/llgo/runtime/compat/purego"
)

const (
	LLGoPackage = "link: -pthread"
	LLGoFiles   = "_wrap/callback.c"
)

//go:linkname callWord C.llgo_test_callback_word
func callWord(fn, value uintptr) uintptr

//go:linkname callIntegers C.llgo_test_callback_integers
func callIntegers(fn uintptr) uintptr

//go:linkname callMixed C.llgo_test_callback_mixed
func callMixed(fn uintptr) uintptr

//go:linkname callNarrow C.llgo_test_callback_narrow
func callNarrow(signed, unsigned, boolean, pointer uintptr, value *uint64) int32

//go:linkname callSort C.llgo_test_callback_sort
func callSort(ascending, descending uintptr) int32

//go:linkname callThread C.llgo_test_callback_thread
func callThread(fn, seed uintptr) int32

//go:linkname callVoid C.llgo_test_callback_void
func callVoid(fn uintptr)

func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func mustPanic(fn func()) {
	recovered := false
	func() {
		defer func() { recovered = recover() != nil }()
		fn()
	}()
	require(recovered, "invalid callback was accepted")
}

var created int

func callback(fn any, marker bool) uintptr {
	pointer := bridge.NewCallback(fn, marker)
	created++
	return pointer
}

func captured(delta uintptr) func(uintptr) uintptr {
	owner := make([]uintptr, 32)
	owner[17] = delta
	return func(value uintptr) uintptr {
		runtime.GC()
		return value + owner[17]
	}
}

func main() {
	for _, invalid := range []any{
		nil, 42, (func())(nil), func(...int) {}, func() (int, int) { return 0, 0 },
		func(string) {}, func([]byte) {}, func(any) {}, func(chan int) {},
		func(complex128) {}, func([2]uintptr) {}, func() float64 { return 0 },
	} {
		mustPanic(func() { bridge.NewCallback(invalid, false) })
	}
	mustPanic(func() { bridge.NewCallback(func(uintptr) {}, true) })

	first := callback(captured(100), false)
	second := callback(captured(200), false)
	require(first != second, "two captured environments share a callback address")
	runtime.GC()
	require(callWord(first, 7) == 107 && callWord(second, 7) == 207, "callback lost captured roots")

	ints := callback(func(a, b, c, d, e, f, g, h, i, j, k, l, m, n, o uintptr) uintptr {
		args := []uintptr{a, b, c, d, e, f, g, h, i, j, k, l, m, n, o}
		var sum uintptr
		for i, value := range args {
			sum += uintptr(i+1) * value
		}
		return sum
	}, false)
	require(callIntegers(ints) == 1240, "15 callback integer arguments")

	mixed := callback(func(a int8, f1 float32, b uint16, f2 float64, c int32, f3 float64,
		d uint64, f4 float64, e uintptr, f5 float64, f int64, f6 float64,
		g uintptr, f7 float64, h uintptr, f8 float64, i uintptr, f9 float64) uintptr {
		require(a == -3 && b == 60001 && c == -1234567 && d == 1<<63+7 && e == 17 && f == -1234567890123,
			"callback integer width/sign extension")
		require(g == 19 && h == 23 && i == 29, "callback integer stack slots")
		require(f1 == 0.25 && f2 == 1.5 && f3 == 2.5 && f4 == 3.5 && f5 == 4.5 &&
			f6 == 5.5 && f7 == 6.5 && f8 == 7.5 && f9 == 8.5, "callback vector and spill arguments")
		return 12345
	}, false)
	require(callMixed(mixed) == 12345, "mixed callback result")

	signed := callback(func(value int8) int8 { require(value == -101, "signed input"); return -91 }, false)
	unsigned := callback(func(value uint16) uint16 { require(value == 60001, "unsigned input"); return 60002 }, false)
	boolean := callback(func(value bool) bool { return !value }, false)
	marker := uint64(0x123456789abcdef0)
	pointer := callback(func(value *uint64) *uint64 {
		require(value == nil || value == &marker, "pointer input")
		return value
	}, false)
	require(callNarrow(signed, unsigned, boolean, pointer, &marker) == 0, "narrow callback returns")

	newComparator := func(direction int) func(*uint64, *uint64) int32 {
		return func(a, b *uint64) int32 {
			if *a < *b {
				return int32(-direction)
			}
			if *a > *b {
				return int32(direction)
			}
			return 0
		}
	}
	ascending := callback(newComparator(1), false)
	descending := callback(newComparator(-1), false)
	require(callSort(ascending, descending) == 0, "two qsort callback closures")

	type CDecl struct{}
	marked := callback(func(_ CDecl, value uintptr) uintptr { return value + 69 }, true)
	require(callWord(marked, 8) == 77, "CDecl marker consumed a C argument")
	voidCalled := false
	callVoid(callback(func() { voidCalled = true }, false))
	require(voidCalled, "void callback")

	var visits int32
	foreign := callback(func(value uintptr) uintptr {
		atomic.AddInt32(&visits, 1)
		return callWord(first, value) + 300
	}, false)
	require(callThread(foreign, 400) == 0 && atomic.LoadInt32(&visits) == 20,
		"foreign-thread GC/nested callback ownership")

	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(value uintptr) {
			defer workers.Done()
			fn := bridge.NewCallback(captured(value), false)
			require(callWord(fn, 10) == value+10, "concurrent callback registration")
		}(uintptr(i))
	}
	workers.Wait()
	created += 16
	for created < 2000 {
		callback(func(value uintptr) uintptr { return value }, false)
	}
	mustPanic(func() { bridge.NewCallback(func() {}, false) })
	runtime.GC()
	require(callWord(first, 11) == 111, "limit rejection invalidated existing callback")
	runtime.KeepAlive(unsafe.Pointer(&marker))
	println("purego callback ABI/GC oracle passed")
}
