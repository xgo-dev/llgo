package main

type Pair struct{ Lo, Hi uint64 }

//go:noinline
func put(p *Pair, x uint64) { p.Hi = x }

//go:noinline
func pointerBranch(op int, x uint64) (r Pair) {
	if op == 0 {
		return Pair{Lo: x}
	}
	put(&r, x)
	return r
}

var saved *Pair

//go:noinline
func escaping(x uint64) (r Pair) {
	saved = &r
	put(&r, x)
	return
}

func deferred(x uint64) (r Pair) {
	defer put(&r, x)
	return
}

func main() {
	r := pointerBranch(0, 17)
	println(r.Lo, r.Hi)
	r = pointerBranch(1, 17)
	println(r.Lo, r.Hi)
	r = deferred(23)
	println(r.Lo, r.Hi)
	var retained []*Pair
	for i := uint64(0); i < 32; i++ {
		escaping(i)
		retained = append(retained, saved)
	}
	var sum uint64
	for _, p := range retained {
		sum += p.Hi
	}
	println(sum)
}
