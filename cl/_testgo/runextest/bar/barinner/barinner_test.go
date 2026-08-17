package barinner_test

import (
	"testing"

	"github.com/xgo-dev/llgo/cl/_testgo/runextest/bar/barinner"
)

func TestBarInner(t *testing.T) {
	if barinner.BarInner() != 2 {
		t.Fatal("BarInner() != 2")
	}
}
