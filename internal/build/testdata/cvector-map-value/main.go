package main

/*
typedef float V __attribute__((vector_size(16)));
*/
import "C"

// Reject by-value nested storage even in an unused declaration.
type Invalid map[int]struct{ V [2]C.V }

func main() {}
