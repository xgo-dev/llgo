package api

type First interface {
	Run() int
}

type Second interface {
	Next() int
}

type Third interface {
	Done() int
}

//go:noinline
func UseFirst(f First) int {
	return f.Run()
}

//go:noinline
func UseSecond(s Second) int {
	return s.Next()
}

//go:noinline
func UseThird(t Third) int {
	return t.Done()
}
