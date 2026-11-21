package offsetdep

//llgo:cdecl
type Record struct {
	Fn   func()
	Tail uint64
}
