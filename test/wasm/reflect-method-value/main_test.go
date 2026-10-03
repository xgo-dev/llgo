package main

import "testing"

// Check the same results with both Go and native LLGo as an oracle for
// the standalone WAMR probe. The WASI regression still builds main.go without
// a testing harness, whose reflection calls could otherwise enable bridges.
func TestMethodValueFixture(t *testing.T) {
	if got, want := methodValues(), [5]int{42, 42, 42, 42, 42}; got != want {
		t.Fatalf("method values = %v, want %v", got, want)
	}
	main()
}
