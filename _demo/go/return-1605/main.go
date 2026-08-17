package main

// Regression test for https://github.com/xgo-dev/llgo/issues/1608

type T struct {
	data []int
}

func a() []int {
	var t = T{data: []int{1, 2}}
	a := t.data
	t.data = []int{1, 2, 3}
	return a
}

func b() ([]int, bool) {
	var t = T{data: []int{1, 2}}
	a := t.data
	t.data = []int{1, 2, 3}
	return a, true
}

func main() {
	resultA := a()
	if len(resultA) != 2 || resultA[0] != 1 || resultA[1] != 2 {
		panic("a(): expect [1,2] but got different")
	}

	resultB, _ := b()
	if len(resultB) != 2 || resultB[0] != 1 || resultB[1] != 2 {
		panic("b(): expect [1,2] but got different")
	}

	println("ok")
}
