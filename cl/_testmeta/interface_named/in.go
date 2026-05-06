package main

type I interface {
	M()
}

type T struct{}

func (T) M() {}

func use(v I) {
	v.M()
}

func main() {
	use(T{})
}
