package main

type Inner struct {
	s string
}

type Outer struct {
	inner Inner
	x     int
}

func (o Outer) Method() string {
	return o.inner.s
}

func main() {
	o := Outer{Inner{"hello"}, 42}
	_ = o.Method()
	var i interface{} = o
	_ = i
}
