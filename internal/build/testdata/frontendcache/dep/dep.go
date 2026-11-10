package dep

import "runtime"

type hidden struct {
	N int
	P *hidden
}
type Alias = hidden

const Third = 1.0 / 3

//go:noinline
func New(n int) *Alias { return &hidden{N: n} }

//go:noinline
func (h *hidden) Value() int { return h.N }

//go:noinline
func Caller() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("caller")
	}
	return file
}

//go:noinline
func Catch(p *int) {
	if recover() != nil {
		*p = 13
	}
}

var dead = map[string]int{"unused": 2}

//llgo:cdecl
type nativeRecord struct {
	Fn   func()
	Tail uint64
}

type Native nativeRecord
type DerivedNative Native

//llgo:cdecl
type NativeAlias = struct {
	Fn   func()
	Tail uint64
}

type DerivedAlias NativeAlias
