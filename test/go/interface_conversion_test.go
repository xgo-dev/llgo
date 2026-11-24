package gotest

import (
	"sync"
	"testing"
)

type conversionSource interface{ Value() int }
type conversionTarget interface{ Value() int }
type conversionA int
type conversionB int

func (v conversionA) Value() int { return int(v) }
func (v conversionB) Value() int { return int(v) }

//go:noinline
func convertInterface(v conversionSource) conversionTarget { return v }

var conversionSink conversionTarget

func TestInterfaceConversionCache(t *testing.T) {
	var existing conversionTarget = conversionA(7)
	var roundTrip conversionSource = existing
	if got := convertInterface(roundTrip); got != existing {
		t.Fatal("round-trip conversion changed the value")
	}
	values := []conversionSource{conversionA(7), conversionB(9), nil, (*conversionPointer)(nil), roundTrip}
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 1000; n++ {
				for _, v := range values {
					got := convertInterface(v)
					if any(got) != any(v) {
						t.Errorf("conversion changed dynamic type or value: %T -> %T", v, got)
						return
					}
					if got != nil && got.Value() != v.Value() {
						t.Error("conversion changed method dispatch")
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

type conversionPointer struct{}

func (*conversionPointer) Value() int { return 11 }

func BenchmarkNamedInterfaceConversion(b *testing.B) {
	b.Run("Monomorphic", func(b *testing.B) {
		var v conversionSource = conversionA(7)
		for n := 0; n < b.N; n++ {
			conversionSink = convertInterface(v)
		}
	})
	b.Run("Alternating", func(b *testing.B) {
		values := [2]conversionSource{conversionA(7), conversionB(9)}
		for n := 0; n < b.N; n++ {
			conversionSink = convertInterface(values[n&1])
		}
	})
}

func TestKnownInterfaceConversion(t *testing.T) {
	var source conversionSource = conversionA(7)
	var target conversionTarget = source
	if target.Value() != 7 || any(target) != any(source) {
		t.Fatal("known value conversion changed its dynamic value")
	}
	if got := (map[conversionTarget]int{target: 1})[conversionA(7)]; got != 1 {
		t.Fatal("known value conversion used the wrong interface identity")
	}
	var nilSource conversionSource = (*conversionPointer)(nil)
	var nilTarget conversionTarget = nilSource
	if nilTarget == nil || any(nilTarget) != any((*conversionPointer)(nil)) || nilTarget.Value() != 11 {
		t.Fatal("known typed nil conversion changed the value or method dispatch")
	}
	var empty any = source
	asserted := empty.(conversionTarget)
	if asserted.Value() != 7 {
		t.Fatal("known value assertion failed")
	}
}
