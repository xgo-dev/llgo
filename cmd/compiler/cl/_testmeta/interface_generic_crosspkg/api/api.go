package api

type I[T any] interface {
	Value() T
}

func UseInt(v I[int]) int {
	return v.Value()
}
