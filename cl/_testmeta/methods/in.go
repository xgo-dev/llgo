package main

type MyType struct {
	x int
}

//go:noinline
func (m MyType) ExportedMethod() string { return "" }

//go:noinline
func (m MyType) unexportedMethod() int { return 0 }

func main() {
	m := MyType{42}
	_ = m.ExportedMethod()
	_ = m.unexportedMethod()
	var i interface{} = m
	_ = i
}
