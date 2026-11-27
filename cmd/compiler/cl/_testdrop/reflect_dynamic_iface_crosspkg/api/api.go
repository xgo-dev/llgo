package api

type Reflector interface {
	ReflectKeep() int
}

var Sink Reflector

//go:noinline
func Accept(r Reflector) {
	Sink = r
}
