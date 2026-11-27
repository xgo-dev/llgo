package api

type Source interface {
	Int63() int64
	Seed(int64)
}

type Source64 interface {
	Source
	Uint64() uint64
}

type Rand struct {
	src Source
	s64 Source64
}

func New(src Source) *Rand {
	s64, _ := src.(Source64)
	return &Rand{src: src, s64: s64}
}

func (r *Rand) Uint64() uint64 {
	if r.s64 != nil {
		return r.s64.Uint64()
	}
	return uint64(r.src.Int63())
}
