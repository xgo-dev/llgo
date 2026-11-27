package main

import (
	"runtime"
	"unsafe"
)

type record struct {
	pad  [64]byte
	a, b int
	data [16]int
}

var garbage any
var sliceSink []int
var stringSink string

//go:noinline
func newRecord(n int) *record { return &record{a: n, b: n + 1} }

//go:noinline
func pressure() {
	for i := 0; i < 3; i++ {
		runtime.GC()
		for j := 0; j < 48; j++ {
			garbage = &record{a: -1, b: -1}
		}
	}
}

//go:noinline
func views(n int) {
	p := newRecord(n)
	p.data[3], p.data[5] = n+2, n+3
	pressure() // Plan the heap owner before deriving views that share its root.
	a, b := &p.a, &p.b
	part := p.data[2:8][1:]
	array := (*[3]int)(part)
	converted := (*int)(unsafe.Pointer(&array[2]))
	p = nil
	pressure()
	if *a != n || *b != n+1 || part[0] != n+2 || *converted != n+3 {
		panic("interior root lost its allocation")
	}
}

//go:noinline
func synthetic(n int) {
	// A nonconstant length with constant capacity uses SSA's synthetic array
	// form. LLGo elides that array and allocates at the slice instead.
	values := make([]int, n, 26)
	// Force the synthetic makeslice allocation onto the heap without keeping
	// a global root alive during the collection below.
	sliceSink = values
	sliceSink = nil
	values[8], values[9] = 81, 92
	part := values[8:10]
	pressure()
	if part[0] != 81 || part[1] != 92 {
		panic("synthetic make slice lost its allocation")
	}
}

//go:noinline
func text(n byte) {
	bytes := []byte{'a', n, n + 1, 'z'}
	str := string(bytes)
	stringSink = str
	stringSink = ""
	sub := str[1:3][1:]
	bytes[2] = 0 // The conversion must keep its independent allocation.
	pressure()
	if sub[0] != n+1 {
		panic("string root lost its allocation")
	}
}

//go:noinline
func closure(n int) func() {
	p := record{a: n, b: n + 1}
	return func() {
		a, b := &p.a, &p.b
		pressure()
		if *a != n || *b != n+1 {
			panic("closure root lost its allocation")
		}
	}
}

//go:noinline
func loop() {
	previous := newRecord(-1)
	p := &previous.a
	for i := 0; i < 5; i++ {
		next := newRecord(i)
		pressure() // Root the new owner independently of the old pointer phi.
		q := &next.a
		pressure()
		if *p != i-1 || *q != i {
			panic("loop root lost an earlier allocation")
		}
		p = q
	}
}

//go:noinline
func unwinding(n int) {
	p := newRecord(n)
	pressure()
	a := &p.a
	defer func() {
		if recover() != n {
			panic("unexpected recovered value")
		}
		pressure()
		if *a != n {
			panic("defer root lost its allocation")
		}
	}()
	pressure()
	panic(n)
}

func main() {
	done := make(chan struct{})
	go func() {
		for i := 0; i < 4; i++ {
			views(40 + i)
			synthetic(12 + i)
			text(byte(40 + i))
			closure(40 + i)()
			unwinding(40 + i)
		}
		loop()
		close(done)
	}()
	<-done
	println("wasm root sharing passed")
}
