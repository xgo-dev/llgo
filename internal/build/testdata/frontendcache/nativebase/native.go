package nativebase

//llgo:cdecl
type Record struct {
	Callback func()
	Tail     uint64
}
