package coveragecgo

import "testing"

func TestCgo(t *testing.T) {
	if got := Add(5); got != 12 {
		t.Fatalf("Add(5) = %d", got)
	}
	if n, err := Failure(); n != -1 || err == nil {
		t.Fatalf("Failure() = %d, %v", n, err)
	}
}
