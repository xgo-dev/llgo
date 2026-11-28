package main

type reading struct {
	Tick  int32
	Seen  [2]bool
	Owner string
}

//go:noinline
func show(r reading) {
	println(r.Tick, r.Seen[0], r.Seen[1], r.Owner)
}

func main() {
	show(reading{17, [2]bool{true, false}, "lab"})
}
