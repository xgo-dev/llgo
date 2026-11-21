// LITTEST
package main

// SYMBOL-NOT: main{{.*}}Wrapper{{.*}}Drop
// SYMBOL-NOT: main{{.*}}struct{{.*}}Drop
// SYMBOL-DAG: main{{.*}}Wrapper{{.*}}Keep
// SYMBOL-DAG: main{{.*}}struct{{.*}}Keep
// SYMBOL-NOT: main{{.*}}Wrapper{{.*}}Drop
// SYMBOL-NOT: main{{.*}}struct{{.*}}Drop

type I interface {
	Keep() int
}

type T struct {
	n int
}

//go:noinline
func (t *T) Keep() int {
	return t.n + 1
}

//go:noinline
func (t *T) Drop() int {
	panic("T.Drop should be unreachable")
}

type Wrapper struct {
	*T
}

type wrapperAlias = struct{ *T }
type wrapperChain = wrapperAlias
type wrapperPointer = *wrapperChain

// This case converts Wrapper, not *T, to a non-empty interface. Wrapper has no
// declared Keep method; its method table slot is a promoted wrapper forwarding
// to (*T).Keep. DCE must keep that wrapper slot alive.
func use(i I) int {
	return i.Keep()
}

func main() {
	println(use(Wrapper{T: &T{n: 41}}))
	// Both aliases must use the anonymous receiver's promoted method symbol.
	value := wrapperChain{T: &T{n: 41}}
	println(use(value))
	println(use(wrapperPointer(&value)))
}
