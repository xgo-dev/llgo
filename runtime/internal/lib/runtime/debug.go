package runtime

func NumCPU() int {
	return int(c_maxprocs())
}

func Breakpoint() {
	c_debugtrap()
}

func Gosched() {
}
