package dep

var Deferred int

//llgo:cold
//llgo:noreturn
//go:noinline
func Stop() {
	defer func() { Deferred++ }()
	panic("stop")
}

//llgo:cold
//llgo:noreturn
//go:noinline
func Generic[T ~int](v T) {
	Stop()
}

type T struct{}

//llgo:cold
//llgo:noreturn
//go:noinline
func (T) Stop() {
	Stop()
}
