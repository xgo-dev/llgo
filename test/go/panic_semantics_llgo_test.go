//go:build llgo

package gotest

import "testing"

func TestNilLoadBeforeBoolShortCircuitPanics(t *testing.T) {
	expectPanicContaining(t, "nil pointer", func() {
		nilLoadBeforeBoolShortCircuit(nil, false)
	})
}

//go:noinline
func nilLoadBeforeBoolShortCircuit(p *int, b bool) int {
	valid := *p >= 0
	if !b || !valid {
		return 5
	}
	return 6
}
