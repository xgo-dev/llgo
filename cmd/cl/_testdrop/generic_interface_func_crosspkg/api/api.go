package api

type I[T any] interface {
	Value() T
}

func Use[T any](v I[T]) T {
	return v.Value()
}
