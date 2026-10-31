package main

/*
typedef float V __attribute__((vector_size(16)));
*/
import "C"

var m = map[C.V]int{C.V{1, 2, 3, 4}: 42}

func main() { println(len(m)) }
