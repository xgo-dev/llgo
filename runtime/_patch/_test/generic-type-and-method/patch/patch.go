package demo

type Box[T any] struct{ Value T }

func (b Box[T]) Old() T { return b.Value }

func Added[T any](v T) T { return v }
