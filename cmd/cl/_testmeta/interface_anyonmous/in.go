package main

type T struct{}

func (T) M() {}
func (T) N() {}

func use(v interface {
	M()
	N()
}) {
	v.M()
	v.N()
}

func main() {
	use(T{})
}
