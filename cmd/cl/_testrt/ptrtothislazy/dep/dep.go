package dep

type Plain struct {
	N int
}

type WithMethod struct {
	N int
}

func (v WithMethod) Value() int {
	return v.N
}

func (*WithMethod) Pointer() int {
	return 42
}
