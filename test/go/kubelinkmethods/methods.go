package kubelinkmethods

type Typed[T comparable] struct {
	items []T
}

func (q *Typed[T]) Add(item T) {
	q.items = append(q.items, item)
}

func (q *Typed[T]) Len() int {
	return len(q.items)
}

type Type = Typed[any]

func NewQueue() *Type {
	return &Typed[any]{}
}

type inner[T any] struct {
	value T
}

func (i *inner[T]) M() {}

func (i *inner[T]) N() int {
	return 1
}

type Outer struct {
	*inner[int]
}

func NewOuter() *Outer {
	return &Outer{inner: &inner[int]{}}
}
