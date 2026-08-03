package a

type T struct{}

func (T) m() string { return "a" }

type I interface{ m() string }
