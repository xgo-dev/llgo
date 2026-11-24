package main

import "example.com/sdkframe"

func main() {
	source := [9]byte{91, 2, 3, 4, 5, 6, 7, 8, 92}
	for _, count := range []uintptr{0, 1, 3, 7} {
		destination := [9]byte{81, 0, 0, 0, 0, 0, 0, 0, 82}
		result := sdkframe.Copy(&destination[1], &source[1], count)
		if result != uint64(count)+37 {
			panic("result")
		}
		for index, value := range destination {
			expected := byte(0)
			if index == 0 {
				expected = 81
			} else if index == 8 {
				expected = 82
			} else if uintptr(index) <= count {
				expected = source[index]
			}
			if value != expected {
				panic("copy or canary")
			}
		}
	}
	println("SDK ABI0 physical frame verified")
}
