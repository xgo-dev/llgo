// LITTEST
package main

// SYMBOL-NOT: main{{.*}}OnlyReader{{.*}}Read
// SYMBOL-NOT: main{{.*}}OnlyReader{{.*}}Drop
// SYMBOL-NOT: main{{.*}}Full{{.*}}Write
// SYMBOL-NOT: main{{.*}}Full{{.*}}Drop
// SYMBOL-DAG: main{{.*}}Full{{.*}}Read
// SYMBOL-NOT: main{{.*}}OnlyReader{{.*}}Read
// SYMBOL-NOT: main{{.*}}OnlyReader{{.*}}Drop
// SYMBOL-NOT: main{{.*}}Full{{.*}}Write
// SYMBOL-NOT: main{{.*}}Full{{.*}}Drop

type Reader interface {
	Read() int
}

type ReadWriter interface {
	Read() int
	Write() int
}

var sink any

// OnlyReader has a Read method and reaches interface-typed metadata through
// any, but it does not implement ReadWriter. A ReadWriter.Read demand must not
// keep OnlyReader.Read alive by name alone.
type OnlyReader struct {
	n int
}

//go:noinline
func (r OnlyReader) Read() int {
	return r.n + 1
}

//go:noinline
func (r OnlyReader) Drop() int {
	panic("OnlyReader.Drop should be unreachable")
}

// Full implements ReadWriter. The only reachable dynamic interface call is
// ReadWriter.Read, so Full.Read remains live while Full.Write and Full.Drop are
// not needed by the final method demand set.
type Full struct {
	n int
}

//go:noinline
func (f Full) Read() int {
	return f.n + 10
}

//go:noinline
func (f Full) Write() int {
	panic("Full.Write should be unreachable")
}

//go:noinline
func (f Full) Drop() int {
	panic("Full.Drop should be unreachable")
}

func use(rw ReadWriter) int {
	return rw.Read()
}

func main() {
	var _ Reader = OnlyReader{}
	sink = OnlyReader{n: 1}
	println(use(Full{n: 32}))
}
