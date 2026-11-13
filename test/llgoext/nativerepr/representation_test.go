package nativerepr

import (
	"testing"
	"unsafe"

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

func increment(x int32) int32 { return x + 1 }

func incrementWord(x uintptr) uintptr { return x + 1 }

func TestNativeWordCallbackDefinitions(t *testing.T) {
	// Memory32 requires a Go/native width adapter even though inheritance
	// retains the source-level signature. Exercise both directions.
	callback := derived.WordCallback(incrementWord)
	if got := callback(41); got != 42 {
		t.Fatalf("derived native word callback = %d, want 42", got)
	}
	goCallback := (func(uintptr) uintptr)(callback)
	if got := goCallback(41); got != 42 {
		t.Fatalf("derived callback converted to Go = %d, want 42", got)
	}
}

func TestNativeCallbackDefinitions(t *testing.T) {
	frame := struct {
		Callback native.Callback
		Sentinel uint64
	}{Callback: increment, Sentinel: 123456}
	p := (*derived.CallbackCopy)(&frame.Callback)
	if got := (*p)(41); got != 42 {
		t.Fatalf("derived callback returned %d, want 42", got)
	}
	*p = nil
	if frame.Callback != nil || frame.Sentinel != 123456 {
		t.Fatal("derived callback write changed adjacent storage")
	}
}

func TestNativeDefinitionArrayStride(t *testing.T) {
	frame := struct {
		Values   [2]derived.Copy
		Sentinel uint64
	}{Values: [2]derived.Copy{{Tail: 7}, {Tail: 11}}, Sentinel: 123456}
	var original [2]native.Record
	if unsafe.Sizeof(frame.Values) != unsafe.Sizeof(original) {
		t.Fatal("derived array uses a different element stride")
	}
	p := (*native.Record)(&frame.Values[1])
	if p.Tail != 11 {
		t.Fatal("second derived element has a different field offset")
	}
	p.Tail = 99
	if frame.Values[0].Tail != 7 || frame.Values[1].Tail != 99 || frame.Sentinel != 123456 {
		t.Fatal("derived array write changed adjacent storage")
	}
}
