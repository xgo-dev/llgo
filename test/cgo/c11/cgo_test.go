package c11

import "testing"

func TestC11Function(t *testing.T) {
	if got := addOne(41); got != 42 {
		t.Fatalf("add_one(41) = %d, want 42", got)
	}
}
