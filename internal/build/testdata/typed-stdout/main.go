package main

import "fmt"

type reading struct {
	Tick  int32
	Seen  [2]bool
	Owner string
}

func main() {
	r := reading{17, [2]bool{true, false}, "lab"}
	n, err := fmt.Printf("%% %v %t\n", r, true)
	if n != len("% {17 [true false] lab} true\n") || err != nil {
		panic("incorrect typed Printf result")
	}
	n, err = fmt.Println(r)
	if n != len("{17 [true false] lab}\n") || err != nil {
		panic("incorrect typed Println result")
	}
}
