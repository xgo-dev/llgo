package crossinline_test

import (
	"testing"

	"github.com/xgo-dev/llgo/benchmark/crossinline/value"
)

// Keep a call boundary as a control for the directly imported method body.
//
//go:noinline
func callMixed(fn func(value.Mixed, int64) value.Mixed, v value.Mixed, delta int64) value.Mixed {
	return fn(v, delta)
}

func TestAggregateMethods(t *testing.T) {
	for delta := range int64(100) {
		pointed := delta + 7
		v := value.Mixed{Integer: 11, Float: 1.25, Pointer: &pointed}
		for i := range v.Payload {
			v.Payload[i] = uint64(i + 3)
		}
		before, want := v, v
		want.Integer += delta
		want.Float += float64(delta)
		want.Payload[7] += uint64(delta)
		direct := v.Shift(delta)
		indirect := callMixed(value.Mixed.Shift, v, delta)
		if direct != want || indirect != want || v != before {
			t.Fatalf("delta %d: receiver copy or aggregate result changed", delta)
		}
		if got := direct.Summary(); got != (value.Summary{Integer: want.Integer, Float: want.Float, Pointer: &pointed}) {
			t.Fatalf("delta %d: summary = %+v", delta, got)
		}
		// Value copies retain pointer identity; the pointee is not deep-copied.
		*direct.Pointer = 42
		if pointed != 42 || indirect.Pointer != &pointed {
			t.Fatal("receiver copy lost pointer identity")
		}
		small := value.ScalarPair{Integer: 11, Float: 1.25}
		if got := small.Shift(delta); got != (value.ScalarPair{Integer: 11 + delta, Float: 1.25 + float64(delta)}) {
			t.Fatalf("delta %d: small result = %+v", delta, got)
		}
	}
}
