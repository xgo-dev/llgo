package model

type Box[T any] struct {
	value T
}

func NewIntBox(v int) *Box[int] {
	return &Box[int]{value: v}
}

func NewUintBox(v uint) *Box[uint] {
	return &Box[uint]{value: v}
}

func NewStringBox(v string) *Box[string] {
	return &Box[string]{value: v}
}

func UseStringBox(v *Box[string]) int {
	return len(v.value)
}

//go:noinline
func (b *Box[T]) Value() T {
	return b.value
}

//go:noinline
func (b *Box[T]) Drop() T {
	panic("Box.Drop should be unreachable")
}
