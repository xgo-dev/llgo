package main

import "runtime"

type box struct {
	p *[64]byte
	n int
}

//go:noinline
func visit(b *box, n int) {
	if n > 0 {
		visit(b, n-1)
	}
	runtime.GC()
	b.n += int(b.p[0])
}

//go:noinline
func local(n int) int {
	var b box
	b.p = new([64]byte)
	b.p[0] = 7
	visit(&b, n)
	return b.n
}

var saved *int

//go:noinline
func escapeA(p *int, n int) {
	if n > 0 {
		escapeB(p, n-1)
	}
	saved = p
}

//go:noinline
func escapeB(p *int, n int) {
	if n > 0 {
		escapeA(p, n-1)
	}
}

//go:noinline
func escaping(n int) {
	var x = n
	escapeB(&x, 2)
}

//go:noinline
func increment(p *int) { *p++ }

func deferred() (r int) {
	var x int
	defer func() { increment(&x); r = x }()
	return
}

func closure() func() int {
	var x int
	increment(&x)
	return func() int { increment(&x); return x }
}

func boxed() any {
	x := 7
	increment(&x)
	return &x
}

func main() {
	println(local(3))
	var retained []*int
	for i := 0; i < 32; i++ {
		escaping(i)
		retained = append(retained, saved)
	}
	runtime.GC()
	sum := 0
	for _, p := range retained {
		sum += *p
	}
	println(sum)
	println(deferred())
	f := closure()
	runtime.GC()
	println(f(), f())
	boxedValue := boxed()
	runtime.GC()
	println(*boxedValue.(*int))
	// A declaration in the loop denotes a distinct object each iteration.
	var previous *int
	distinct := true
	for i := 0; i < 10; i++ {
		var x int
		increment(&x)
		if previous == &x {
			distinct = false
		}
		previous = &x
	}
	println(distinct)
}
