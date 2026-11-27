// LITTEST
package main

// SYMBOL-NOT: main{{.*}}T{{.*}}Drop
// SYMBOL-DAG: main{{.*}}T{{.*}}Keep
// SYMBOL-NOT: main{{.*}}T{{.*}}Drop

var sink any
var keepMethod = T.Keep

// This case keeps T's runtime type metadata reachable through an empty
// interface conversion. T.Drop must still be removed because no interface or
// direct call path requires that method slot.
type T struct {
	n int
}

//go:noinline
func (t T) Keep() int {
	return t.n + 1
}

//go:noinline
func (t T) Drop() int {
	panic("Drop should be unreachable")
}

func main() {
	t := T{n: 41}
	sink = t
	println(keepMethod(t))
}
