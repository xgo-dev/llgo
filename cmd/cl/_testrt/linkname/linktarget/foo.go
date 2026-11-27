package linktarget

import _ "unsafe"

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

func F(a, b *int8) {
	printf(cstr("a: %s, b: %s\n"), a, b)
}

var _ m

type m struct {
	s string
}

func (t m) info() string {
	return t.s
}

func (t *m) setInfo(s string) {
	t.s = s
}
