package helper

func Next(n int) int { return n + 1 }

//go:noinline
func Fail() { panic("runtime-free cache sentinel") }
