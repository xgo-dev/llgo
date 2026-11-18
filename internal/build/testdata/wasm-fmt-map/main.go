package main

import "fmt"

func main() {
	values := map[string]int{"世界": 42, "b": -7, "a": 0, "": 8}
	if got := fmt.Sprintf("%v", values); got != "map[:8 a:0 b:-7 世界:42]" {
		panic(got)
	}
	var empty map[string]int
	if got := fmt.Sprintf("%v", empty); got != "map[]" {
		panic(got)
	}
	println("wasi map format ok")
}
