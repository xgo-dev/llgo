package gotest

import (
	"testing"
	"unsafe"
)

func TestStringEmptySuffixKeepsBase(t *testing.T) {
	s := stringSliceBaseValue()
	base := stringDataBase(s)
	n := len(s)

	tail := s[n-1:]
	cases := []struct {
		name string
		got  string
	}{
		{"s[n:]", s[n:]},
		{"s[n:n]", s[n:n]},
		{"s[1:][2:][2:]", s[1:][2:][2:]},
		{"s[n-1:][1:]", tail[1:]},
		{"s[n-1:][1:len(s[n-1:])]", tail[1:len(tail)]},
	}
	for _, tc := range cases {
		gotBase := stringDataBase(tc.got)
		if gotBase-base >= uintptr(len(s)) {
			t.Errorf("%s data base = %#x, want within original string [%#x, %#x)", tc.name, gotBase, base, base+uintptr(len(s)))
		}
	}
}

func stringSliceBaseValue() string {
	return string([]byte{'h', 'e', 'l', 'l', 'o'})
}

func stringDataBase(s string) uintptr {
	return *(*uintptr)(unsafe.Pointer(&s))
}
