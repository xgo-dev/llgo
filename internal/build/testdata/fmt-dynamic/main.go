package main

import (
	"fmt"
	"os"
)

type item struct{ Text string }

func (i item) String() string { return "item=" + i.Text }

func main() {
	println(fmt.Sprintf("%v", map[string]int{"b": 2, "a": 1}))
	println(fmt.Sprintf("%v", []item{{"one"}, {"two"}}))
	formats := []string{
		"%v", "%.1f", "%#v", "%[2]*.[3]*[1]f", "%*[2]v", "%*.*f",
		"%T/%T", "%p", "%[0]d", "%[9]x", "%w", "%q", "%", "",
		"%% %v %v %v %v", "%[4]+g", "%020.2f", "%s", "%#x",
	}
	if len(os.Args) > 1 {
		formats = os.Args[1:]
	}
	for _, format := range formats {
		println(fmt.Sprintf(format, 42.125, 8, 2, complex(1, -2), "世界", true, uintptr(6)))
	}
}
