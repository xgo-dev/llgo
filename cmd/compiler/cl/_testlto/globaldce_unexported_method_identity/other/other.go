package other

type Other struct{}

//go:noinline
func (Other) hidden() int {
	panic("Other.hidden should be unreachable")
}
