package factory

import "github.com/xgo-dev/llgo/cl/_testdrop/iface_flow_crosspkg/api"

type T struct {
	n int
}

//go:noinline
func (t T) Keep() int {
	return t.n + 1
}

//go:noinline
func (t T) Drop() int {
	panic("Drop should be unreachable")
}

func Make(n int) api.Keeper {
	return T{n: n}
}
