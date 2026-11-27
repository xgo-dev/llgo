//go:build go1.27

package main

type Accumulator struct {
	base int
}

type Stringer interface {
	String() string
}

func (a Accumulator) Add[T ~int](value T) int {
	return a.base + int(value)
}

func (a Accumulator) Keep[T any](value T) T {
	return value
}

func (a Accumulator) String() string {
	return "accumulator"
}

func main() {
	a := Accumulator{base: 40}
	add := a.Add[int]
	keep := Accumulator.Keep[string]
	var stringer Stringer = a
	println(a.Add(2), add(3), keep(a, "generic-method"), stringer.String())
}
