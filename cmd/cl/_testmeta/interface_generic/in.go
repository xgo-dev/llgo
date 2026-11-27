package main

type Box[T any] struct {
	value T
}

func (b *Box[T]) Value() T {
	return b.value
}

type I[T any] interface {
	Value() T
}

func useInt(v I[int]) int {
	return v.Value()
}

func main() {
	_ = useInt(&Box[int]{value: 42})
}
