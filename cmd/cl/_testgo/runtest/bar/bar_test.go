package bar

import "testing"

func TestBar(t *testing.T) {
	if Bar() != 2 {
		t.Fatal("Bar() != 2")
	}
}
