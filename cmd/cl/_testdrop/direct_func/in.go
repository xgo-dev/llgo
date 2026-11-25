// LITTEST
package main

// SYMBOL-NOT: main{{.*}}Drop
// SYMBOL-DAG: main{{.*}}Keep
// SYMBOL-NOT: main{{.*}}Drop

var keepFunc = Keep

//go:noinline
func Keep() int {
	return 42
}

//go:noinline
func Drop() int {
	panic("Drop should be unreachable")
}

func main() {
	println(keepFunc())
}
