package gotest

import (
	"runtime"
	"testing"
)

type nestedPointerValue struct{ value int }

//go:noinline
func nestedPointerRestore(p *nestedPointerValue, call func()) func() int {
	return func() int {
		previous := p.value
		defer func() { p.value = previous }()
		p.value++
		call()
		return p.value
	}
}

//go:noinline
func nestedPointerPair(p, q *nestedPointerValue) func() int {
	return func() int {
		previousP, previousQ := p.value, q.value
		defer func() { p.value, q.value = previousP, previousQ }()
		p.value++
		q.value += 2
		return p.value + q.value
	}
}

func TestNestedPointerCapturePointee(t *testing.T) {
	p, q := &nestedPointerValue{7}, &nestedPointerValue{11}
	call := nestedPointerRestore(p, func() {})
	for _, value := range []int{7, 19, 31} {
		p.value = value
		if got := call(); got != value+1 || p.value != value {
			t.Fatalf("capture copied the pointee: result=%d, pointee=%d, want %d, %d", got, p.value, value+1, value)
		}
	}
	p.value = 7
	pair := nestedPointerPair(p, q)
	if got := pair(); got != 21 || p.value != 7 || q.value != 11 {
		t.Fatalf("two captures: result=%d, pointees=%d,%d", got, p.value, q.value)
	}
	panicking := nestedPointerRestore(p, func() { panic("callback") })
	func() {
		defer func() {
			if recover() != "callback" || p.value != 7 {
				t.Fatal("deferred pointee restore was lost during panic")
			}
		}()
		panicking()
		t.Fatal("callback did not panic")
	}()
}

//go:noinline
func nestedPointerReassign(p, q *nestedPointerValue) func() int {
	return func() int {
		defer func() { p = q }()
		return p.value
	}
}

func TestNestedPointerCaptureSharedBinding(t *testing.T) {
	p, q := &nestedPointerValue{7}, &nestedPointerValue{23}
	call := nestedPointerReassign(p, q)
	if first, second := call(), call(); first != 7 || second != 23 {
		t.Fatalf("nested reassignment: %d,%d, want 7,23", first, second)
	}
	var address **nestedPointerValue
	read := func() int {
		defer func() { address = &p }()
		return p.value
	}
	if read() != 7 || address != &p {
		t.Fatal("captured variable lost its address identity")
	}
	*address = q
	if read() != 23 {
		t.Fatal("write through captured variable's address was lost")
	}
	read, replace := func(p *nestedPointerValue) (func() int, func(*nestedPointerValue)) {
		return func() int {
			defer func() { _ = p.value }()
			return p.value
		}, func(next *nestedPointerValue) { p = next }
	}(p)
	replace(&nestedPointerValue{41})
	if read() != 41 {
		t.Fatal("a sibling closure no longer shares the binding")
	}
}

//go:noinline
func nestedPointerHeap(value int) func() int {
	return nestedPointerRestore(&nestedPointerValue{value}, runtime.GC)
}

//go:noinline
func nestedPointerForward(p *nestedPointerValue) func() func() int {
	return func() func() int {
		return func() int {
			p.value++
			return p.value
		}
	}
}

func TestNestedPointerCaptureGCAndLoop(t *testing.T) {
	outer := nestedPointerForward(&nestedPointerValue{17})
	inner := outer()
	outer = nil
	runtime.GC()
	if first, second := inner(), inner(); first != 18 || second != 19 {
		t.Fatalf("returned inner closure: %d,%d, want 18,19", first, second)
	}
	for i := 0; i < 8; i++ {
		call := nestedPointerHeap(i)
		runtime.GC()
		if first, second := call(), call(); first != i+1 || second != i+1 {
			t.Fatalf("captured heap object: %d,%d, want %d", first, second, i+1)
		}
	}
	values := [...]nestedPointerValue{{3}, {5}, {9}}
	var calls [3]func() int
	for i := range values {
		p := &values[i]
		calls[i] = func() int {
			previous := p.value
			defer func() { p.value = previous }()
			p.value++
			return p.value
		}
	}
	runtime.GC()
	for i, call := range calls {
		if got := call(); got != values[i].value+1 {
			t.Fatalf("loop capture %d: got %d, pointee %d", i, got, values[i].value)
		}
	}
}
