package escapephase1

func read(*int) int {
	return 7
}

func Local() int {
	p := new(int)
	return read(p)
}
