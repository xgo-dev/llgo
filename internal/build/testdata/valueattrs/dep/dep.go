package dep

//go:noinline
//llgo:result nonnull sameas(p)
func Checked(p *int) *int {
	if p == nil {
		panic("nil")
	}
	return p
}
