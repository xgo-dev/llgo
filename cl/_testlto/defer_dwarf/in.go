package main

import "fmt"

func record(values *[]int, value int) {
	*values = append(*values, value)
}

func deferredValues(addSecond bool) (values []int) {
	defer record(&values, 1)
	if addSecond {
		defer record(&values, 2)
	}
	return
}

func main() {
	fmt.Println(deferredValues(true))
}
