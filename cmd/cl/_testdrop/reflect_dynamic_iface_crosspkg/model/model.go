package model

type Used struct {
	n int
}

func NewUsed(n int) Used {
	return Used{n: n}
}

//go:noinline
func (u Used) ReflectKeep() int {
	return u.n + 1
}

type Unused struct {
	n int
}

func NewUnused(n int) Unused {
	return Unused{n: n}
}

//go:noinline
func (u Unused) ReflectKeep() int {
	panic("Unused.ReflectKeep should be unreachable")
}

func UseUnused(u Unused) int {
	return u.n
}
