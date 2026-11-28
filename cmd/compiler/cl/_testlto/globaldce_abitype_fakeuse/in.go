package main

type Key struct {
	A int
	B string
}

type Value struct {
	N int
}

func main() {
	k := Key{A: 1, B: "x"}
	var v any = k
	if got, ok := v.(Key); !ok || got != k {
		panic("interface key assertion failed")
	}

	m := map[Key]Value{k: {N: 7}}
	var mv any = m
	if mv == nil {
		panic("map interface is nil")
	}
	if got := m[k].N; got != 7 {
		panic("map lookup failed")
	}
	println("pass")
}
