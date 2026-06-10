package main

type Stringer interface {
	String() string
}

type MyType struct {
	x int
}

func (m MyType) String() string {
	return "hello"
}

func useIface(s Stringer) {
	s.String()
}

func main() {
	var s Stringer = MyType{x: 42}
	useIface(s)
}
