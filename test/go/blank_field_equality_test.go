package gotest

import (
	"testing"
	"unsafe"
)

type blankFieldComparable struct {
	A uintptr
	_ uintptr
	B uintptr
}

type blankFieldLayout struct {
	A uintptr
	H uintptr
	B uintptr
}

func TestBlankFieldIgnoredInStructEquality(t *testing.T) {
	x := blankFieldComparable{A: 1, B: 2}
	y := blankFieldComparable{A: 1, B: 2}

	(*blankFieldLayout)(unsafe.Pointer(&x)).H = 3
	(*blankFieldLayout)(unsafe.Pointer(&y)).H = 9

	if x != y {
		t.Fatalf("blank fields should be ignored in equality: %#v != %#v", x, y)
	}
}
