package main

import "fmt"

func main() {
	text := "first\n"
	n, err := fmt.Printf("first\n")
	if err != nil || n != len(text) {
		panic("stdout write result changed")
	}
	fmt.Printf("checked\n")
}
