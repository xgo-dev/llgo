package main

import "strconv"

func main() {
	value, err := strconv.Atoi("42")
	if err != nil || value != 42 {
		panic("bad conversion")
	}
	println(value)
}
