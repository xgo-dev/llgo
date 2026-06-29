//go:build go1.23
// +build go1.23

package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

func main() {
	// Existing array or memory region
	var arr [5]int = [5]int{10, 20, 30, 40, 50}

	// Use SliceAt to create a slice in the middle of the array (zero allocation)
	// Starting from arr[1] with length 3
	sliceVal := reflect.SliceAt(
		reflect.TypeOf(0),       // Element type: int
		unsafe.Pointer(&arr[1]), // Base address
		3,                       // Length
	)

	slice := sliceVal.Interface().([]int)
	if r := fmt.Sprint(slice); r != "[20 30 40]" {
		panic("error: " + r)
	}

	// Modifying the slice affects the original array
	slice[0] = 999
	if r := fmt.Sprint(arr); r != "[10 999 30 40 50]" {
		panic("error: " + r)
	}
}
