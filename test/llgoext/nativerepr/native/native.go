package native

//llgo:cdecl
type Record struct {
	Fn   func()
	Tail uint64
}

func (Record) NativeMethod() {}

//llgo:cdecl
type Generic[T any] struct {
	Fn   func()
	Tail T
}

//llgo:cdecl
type Recursive struct {
	Fn   func()
	Tail uint64
	Next *Recursive
}
