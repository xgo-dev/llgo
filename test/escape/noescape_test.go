package escape

import (
	"testing"
	"unsafe"
)

//go:linkname trustedNoEscape llgo.noescape
func trustedNoEscape(p unsafe.Pointer) unsafe.Pointer {
	return p
}

func TestNoEscapePreservesPointerAndEvaluation(t *testing.T) {
	x, calls := 17, 0
	argument := func() unsafe.Pointer {
		calls++
		return unsafe.Pointer(&x)
	}
	p := trustedNoEscape(argument())
	if calls != 1 || p != unsafe.Pointer(&x) {
		t.Fatalf("identity/evaluation: calls=%d pointer=%p", calls, p)
	}
	*(*int)(p) = 29
	if x != 29 || trustedNoEscape(nil) != nil {
		t.Fatal("pointer identity or nil changed")
	}
}

func TestNoEscapeDoesNotHideByteSliceMutation(t *testing.T) {
	s := "immutable"
	b := []byte(s)
	p := trustedNoEscape(unsafe.Pointer(&b[0]))
	*(*byte)(p) = 'I'
	if s != "immutable" || string(b) != "Immutable" {
		t.Fatalf("mutated string storage: string=%q bytes=%q", s, b)
	}
}
