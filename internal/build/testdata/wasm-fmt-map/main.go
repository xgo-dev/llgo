package main

import "fmt"

type score uint16

type key struct {
	Name string
	IDs  [2]int
}

func main() {
	values := map[string]int{"世界": 42, "b": -7, "a": 0, "": 8}
	if got := fmt.Sprintf("%v", values); got != "map[:8 a:0 b:-7 世界:42]" {
		panic(got)
	}
	var empty map[string]int
	if got := fmt.Sprintf("%v", empty); got != "map[]" {
		panic(got)
	}
	nested := map[key][]score{
		{Name: "b"}:                    {4},
		{Name: "a", IDs: [2]int{1, 2}}: {2, 3},
		{Name: "a"}:                    nil,
	}
	if got := fmt.Sprintf("%v", nested); got != "map[{a [0 0]}:[] {a [1 2]}:[2 3] {b [0 0]}:[4]]" {
		panic(got)
	}
	aggregate := struct {
		Values []complex64
		Flags  [2]bool
	}{[]complex64{1 - 2i, -3 + 4i}, [2]bool{true, false}}
	if got := fmt.Sprintf("%v", &aggregate); got != "&{[(1-2i) (-3+4i)] [true false]}" {
		panic(got)
	}
	println("wasi default value formatting ok")
}
