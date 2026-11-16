// Package value supplies small cross-package calls for the inline benchmarks.
package value

func Mix(state, x uint64) uint64 {
	return (state ^ x) * 0x9e3779b97f4a7c15
}

type Number uint64

func (n Number) Mix(x uint64) Number {
	return Number(Mix(uint64(n), x))
}

type State struct {
	Value uint64
}

func (s *State) Mix(x uint64) {
	s.Value = Mix(s.Value, x)
}

type Pair struct {
	Left, Right uint64
}

func Sum(p Pair) uint64 {
	return p.Left + p.Right
}

func (p Pair) Sum() uint64 {
	return p.Left + p.Right
}

func (p *Pair) PointerSum() uint64 {
	return p.Left + p.Right
}

type Record struct {
	Fields [8]uint64
}

func First(r Record) uint64 {
	return r.Fields[0]
}

func (r Record) First() uint64 {
	return r.Fields[0]
}
