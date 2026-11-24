package main

import "fmt"

type reading struct {
	Tick  int32
	Seen  [2]bool
	Owner string
}

//go:noinline
func show(r reading) {
	fmt.Printf("%v\n", r)
}

func main() {
	show(reading{17, [2]bool{true, false}, "lab"})
}
