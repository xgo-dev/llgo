package gotest

import "testing"

type assertionEmpty interface{}
type assertionSource interface{ EmptyAssertValue() int }
type assertionValue int

func (v assertionValue) EmptyAssertValue() int { return int(v) }

//go:noinline
func assertEmpty(v assertionSource) (any, bool) { result, ok := v.(any); return result, ok }

//go:noinline
func assertNamedEmpty(v assertionSource) (any, bool) {
	result, ok := v.(assertionEmpty)
	return result, ok
}

//go:noinline
func assertEmptyDirect(v assertionSource) any { return v.(assertionEmpty) }

func TestEmptyInterfaceAssertion(t *testing.T) {
	for _, value := range []assertionSource{nil, assertionValue(7), (*assertionValue)(nil)} {
		for _, assert := range []func(assertionSource) (any, bool){assertEmpty, assertNamedEmpty} {
			got, ok := assert(value)
			if ok != (value != nil) || got != any(value) {
				t.Fatalf("assert %T: got (%v, %v), want (%v, %v)", value, got, ok, value, value != nil)
			}
		}
		if value != nil && assertEmptyDirect(value) != any(value) {
			t.Fatal("direct empty-interface assertion changed the dynamic value")
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("nil direct assertion did not panic")
		}
	}()
	assertEmptyDirect(nil)
}
