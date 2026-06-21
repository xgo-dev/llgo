//go:build linux || darwin

package gotest

import (
	"reflect"
	"syscall"
	"testing"
	"unsafe"
)

type compareStringInt struct {
	S string
	I int
}

type compareStringString struct {
	S string
	T string
}

type compareInterfaceThenScalar struct {
	V any
	N int
}

func TestInterfaceStructStringCompareShortCircuits(t *testing.T) {
	bad1, bad2, cleanup := protectedStrings(t)
	defer cleanup()

	cases := []struct {
		name string
		a    any
		b    any
	}{
		{
			name: "scalar field differs after string",
			a:    compareStringInt{S: bad1, I: 1},
			b:    compareStringInt{S: bad2, I: 2},
		},
		{
			name: "later string length differs",
			a:    compareStringString{S: bad1, T: "a"},
			b:    compareStringString{S: bad2, T: "aa"},
		},
		{
			name: "earlier safe string differs",
			a:    compareStringString{S: "a", T: bad1},
			b:    compareStringString{S: "b", T: bad2},
		},
	}
	for _, tc := range cases {
		if tc.a == tc.b {
			t.Fatalf("%s: interface struct comparison returned true", tc.name)
		}
	}
}

func TestInterfaceStructComparePreservesInterfacePanicOrder(t *testing.T) {
	expectPanicContaining(t, "comparing uncomparable type []int", func() {
		_ = compareInterfaceThenScalar{V: []int{}, N: 1} == compareInterfaceThenScalar{V: []int{}, N: 2}
	})
}

func protectedStrings(t *testing.T) (string, string, func()) {
	t.Helper()

	pageSize := syscall.Getpagesize()
	page, err := syscall.Mmap(-1, 0, pageSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Skipf("mmap unavailable: %v", err)
	}
	if err := syscall.Mprotect(page, syscall.PROT_NONE); err != nil {
		_ = syscall.Munmap(page)
		t.Skipf("mprotect unavailable: %v", err)
	}

	bad1 := "foo"
	bad2 := "foo"
	(*reflect.StringHeader)(unsafe.Pointer(&bad1)).Data = uintptr(unsafe.Pointer(&page[0]))
	(*reflect.StringHeader)(unsafe.Pointer(&bad2)).Data = uintptr(unsafe.Pointer(&page[1]))

	return bad1, bad2, func() {
		_ = syscall.Munmap(page)
	}
}
