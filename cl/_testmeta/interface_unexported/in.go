package main

type I interface {
	m()
}

type T struct{}

func (T) m() {}

func use(v I) {
	v.m()
}

func main() {
	use(T{})
}
