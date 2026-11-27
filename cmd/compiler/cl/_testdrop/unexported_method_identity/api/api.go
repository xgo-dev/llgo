package api

type hiddenIface interface {
	hidden() int
}

type Good struct {
	n int
}

//go:noinline
func (g Good) hidden() int {
	return g.n + 1
}

func NewGood(n int) Good {
	return Good{n: n}
}

func Use(v hiddenIface) int {
	return v.hidden()
}
