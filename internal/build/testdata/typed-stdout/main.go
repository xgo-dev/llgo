package main

import (
	"fmt"
	"syscall"
)

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
	if n, err := printValue("%[1]g\n", 42.125); n != len("42.125\n") || err != nil {
		panic("incorrect dynamic format write result")
	}
	if n, err := printValue("%v\n", "value"); n != len("value\n") || err != nil {
		panic("incorrect dynamic operand write result")
	}
	if err := syscall.Close(1); err != nil {
		panic(err)
	}
	if n, err := printValue("%s", "closed"); n != 0 || err == nil {
		panic("incorrect dynamic write error")
	}

}

//go:noinline
func printValue(format string, value any) (int, error) {
	return fmt.Printf(format, value)
}
