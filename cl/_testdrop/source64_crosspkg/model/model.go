package model

import "github.com/xgo-dev/llgo/cl/_testdrop/source64_crosspkg/api"

type RuntimeSource struct {
	n uint64
}

func NewRuntimeSource(n uint64) api.Source {
	return &RuntimeSource{n: n}
}

//go:noinline
func (s *RuntimeSource) Int63() int64 {
	return int64(s.n)
}

//go:noinline
func (s *RuntimeSource) Seed(seed int64) {
	s.n = uint64(seed)
}

//go:noinline
func (s *RuntimeSource) Uint64() uint64 {
	return s.n + 1
}

//go:noinline
func (s *RuntimeSource) Drop() uint64 {
	panic("RuntimeSource.Drop should be unreachable")
}

type Uint64Only struct {
	n uint64
}

func NewUint64Only(n uint64) Uint64Only {
	return Uint64Only{n: n}
}

func UseUint64Only(v Uint64Only) uint64 {
	return v.n
}

//go:noinline
func (v Uint64Only) Uint64() uint64 {
	panic("Uint64Only.Uint64 should be unreachable")
}
