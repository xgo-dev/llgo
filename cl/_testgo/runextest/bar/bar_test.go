package bar_test

import (
	"testing"

	"github.com/xgo-dev/llgo/cl/_testgo/runextest/bar"
)

func TestBar(t *testing.T) {
	if bar.Bar() != 2 {
		t.Fatal("Bar() != 2")
	}
}
