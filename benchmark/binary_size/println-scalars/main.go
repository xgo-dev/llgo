package main

//go:noinline
func show(i int64, u uint64, ok bool, s string) {
	println(i, u, ok, s)
}

func main() {
	show(-17, 42, true, "lab")
}
