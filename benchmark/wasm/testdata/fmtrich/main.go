package main

import "fmt"

type sample struct {
	N int
	S string
}
type label string

func (l label) String() string { return "label<" + string(l) + ">" }

func main() {
	v := sample{N: 7, S: "go"}
	fmt.Printf(
		"bool=%t int=%+08d hex=%#x oct=%#o float=%8.3f sci=%e str=%q bytes=% x rune=%U type=%T struct=%+v go=%#v slice=%v map=%v stringer=%s\n",
		true, -42, uint(42), uint(42), 3.5, 3.5, "世界", []byte{1, 2, 255}, '界', v, v, v, []int{1, 2, 3}, map[string]int{"b": 2, "a": 1}, label("ok"),
	)
}
