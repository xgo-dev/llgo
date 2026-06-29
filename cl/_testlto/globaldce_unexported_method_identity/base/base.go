package base

type hiddenIface interface {
	hidden() int
}

type Exported struct{}

//go:noinline
func (Exported) hidden() int {
	return 17
}

func Call(v hiddenIface) int {
	return v.hidden()
}
