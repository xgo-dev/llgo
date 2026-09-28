package test

import "testing"

type attributeContainer struct {
	P       *int
	N       uint32
	Payload [64]byte
}

//go:noinline
//llgo:param(p) nonnull
//llgo:result(pointer) nonnull sameas(p)
//llgo:result(count) range(0,64)
func attributeAggregate(input attributeContainer, p *int, n uint32) (pointer *int, count uint32, result attributeContainer) {
	input.P = p
	input.N = n & 63
	return p, input.N, input
}

//go:noinline
//llgo:result(0) sameas(p)
func attributeEntrySnapshot(p *int, source *attributeContainer, replacement *int) *int {
	source.P = replacement
	return p
}

type attributePacked struct {
	Signed int8
	Count  uint8
	Bytes  [6]byte
}

//go:noinline
//llgo:param(n) range(-3,5)
//llgo:result(signed) sameas(n)
//llgo:result(count) range(0,64)
func attributePackedRoundTrip(input attributePacked, n int8) (signed int8, count uint8, result attributePacked) {
	input.Signed = n
	input.Count &= 63
	return n, input.Count, input
}

type attributeLarge struct {
	P       *int
	Payload [10000]uint64
	Count   uint32
}

//go:noinline
//llgo:param(p) nonnull
func attributeLargeResult(p *int) (out attributeLarge) {
	out.P = p
	out.Payload[0], out.Payload[9999] = 19, 101
	out.Count = 7
	return
}

func TestSourceContractsAfterABI(t *testing.T) {
	value := 17
	in := attributeContainer{P: &value, N: 999, Payload: [64]byte{0: 1, 63: 2}}
	for n := uint32(0); n < 130; n++ {
		p, count, out := attributeAggregate(in, &value, n)
		if p != &value || count != n%64 || out.P != &value || out.N != n%64 || out.Payload != in.Payload {
			t.Fatal("indirect arguments or multiple results changed across ABI conversion")
		}
		if in.N != 999 {
			t.Fatal("callee changed the caller's by-value input")
		}
	}
	replacement := 29
	old := attributeEntrySnapshot(in.P, &in, &replacement)
	if old != &value || in.P != &replacement {
		t.Fatal("sameas reloaded changed input memory")
	}
	for n := int8(-3); n < 5; n++ {
		in := attributePacked{Signed: n, Count: 193, Bytes: [6]byte{0: 11, 5: 253}}
		signed, count, out := attributePackedRoundTrip(in, n)
		if signed != n || count != 1 || out.Signed != n || out.Count != 1 || out.Bytes != in.Bytes {
			t.Fatalf("packed argument or multiple results changed unrelated data: %#v", out)
		}
	}
	large := attributeLargeResult(&value)
	if large.P != &value || large.Count != 7 || large.Payload[0] != 19 || large.Payload[9999] != 101 {
		t.Fatal("large indirect result lost values")
	}
}

//go:noinline
//llgo:result(out) sameas(p)
func deferredResultRelation(p *int) (out *int, n int) {
	defer func() {
		if recover() != nil {
			out = p
			n = 17
		}
	}()
	panic("deferred result")
}

func TestResultRelationAfterDeferredRecovery(t *testing.T) {
	x := 42
	if p, n := deferredResultRelation(&x); p != &x || n != 17 {
		t.Fatalf("deferred result: %p %d", p, n)
	}
}
