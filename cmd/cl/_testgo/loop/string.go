package main

func zStringLen(s string) int { return len(s) }

// A small fixed trip count preserves the constant-string no-alloc regression;
// the former ten-million-iteration runtime load added no lowering coverage.
func zStringLoop() int {
	total := 0
	for i := 0; i < 4; i++ {
		total += zStringLen("hello")
	}
	return total
}
