package nativerepr

import (
	"testing"

	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/derived"
	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/native"
)

type localCopy derived.Copy

func TestNativeDefinitionPointerConversion(t *testing.T) {
	h := struct {
		Value    native.Record
		Sentinel uint64
	}{Value: native.Record{Tail: 7}, Sentinel: 123456}
	p := (*localCopy)(&h.Value)
	before := p.Tail
	p.Tail = 99
	if before != 7 || h.Value.Tail != 99 || h.Sentinel != 123456 {
		t.Fatalf("read/value/sentinel = %d/%d/%d, want 7/99/123456", before, h.Value.Tail, h.Sentinel)
	}
	q := (*native.Record)(p)
	q.Tail = 100
	if p.Tail != 100 || h.Sentinel != 123456 {
		t.Fatal("reverse conversion lost shared storage")
	}
	alias := (*derived.ThroughAlias)(&h.Value)
	alias.Tail = 101
	if h.Value.Tail != 101 || h.Sentinel != 123456 {
		t.Fatal("definition through alias lost shared storage")
	}
}

func TestNativeGenericAndRecursiveDefinitions(t *testing.T) {
	g := struct {
		Value    native.Generic[uint64]
		Sentinel uint64
	}{Value: native.Generic[uint64]{Tail: 7}, Sentinel: 123456}
	p := (*derived.Instance)(&g.Value)
	before := p.Tail
	p.Tail = 99
	if before != 7 || g.Value.Tail != 99 || g.Sentinel != 123456 {
		t.Fatalf("generic read/value/sentinel = %d/%d/%d", before, g.Value.Tail, g.Sentinel)
	}
	r := native.Recursive{Tail: 7}
	r.Next = &r
	q := (*derived.Recursive)(&r)
	if q.Tail != 7 || q.Next != &r {
		t.Fatal("recursive definition changed field locations")
	}
}

//go:noinline
func callGoClosure(fn func(derived.Copy) uint64, value derived.Copy) uint64 {
	return fn(value)
}

func TestGoClosureAcceptsNativeDerivedValue(t *testing.T) {
	marker := uint64(30)
	fn := func(value derived.Copy) uint64 { return value.Tail + marker }
	if got := callGoClosure(fn, derived.Copy{Tail: 7}); got != 37 {
		t.Fatalf("ordinary Go closure call = %d, want 37", got)
	}
}
