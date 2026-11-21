// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

type Pair[K comparable, V any] struct {
	Key K
	Val V
}

type Stack[T any] []T

func (s *Stack[T]) Push(v T) { *s = append(*s, v) }
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(*s) == 0 {
		return zero, false
	}
	old := *s
	v := old[len(old)-1]
	*s = old[:len(old)-1]
	return v, true
}

type Number interface {
	~int | ~float64
}

func Sum[T Number](xs []T) T {
	var t T
	for _, x := range xs {
		t += x
	}
	return t
}

func Map[T, U any](xs []T, f func(T) U) []U {
	out := make([]U, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

func main() {
	var st Stack[string]
	st.Push("a")
	st.Push("b")
	v, ok := st.Pop()
	println("pop", v, ok, len(st))
	p := Pair[string, int]{"k", 42}
	println("pair", p.Key, p.Val)
	println("sum", Sum([]int{1, 2, 3}))
	d := Sum([]float64{1.5, 2.25})
	println("sumf", int(d*100))
	m := Map([]int{1, 2}, func(x int) string { return "n" })
	println("map", len(m), m[0])
	var q Stack[Stack[int]]
	q = append(q, Stack[int]{1, 2})
	println("nested", len(q), len(q[0]), q[0][1])
	f := st.Pop
	v2, ok2 := f()
	println("mv", v2, ok2, len(st))
	c := Pair[string, Stack[int]]{"s", Stack[int]{9}}
	println("nestpair", c.Key, len(c.Val), c.Val[0])
}
