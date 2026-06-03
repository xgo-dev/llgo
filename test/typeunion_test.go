package test

import "testing"

type PublicKey any

type VerificationKey interface {
	PublicKey | []uint8
}

type VerificationKeySet struct {
	Keys []VerificationKey
}

func checkVerificationKey(got any) int {
	switch have := got.(type) {
	case VerificationKeySet:
		return len(have.Keys)

	case VerificationKey:

		_ = have
		return 100

	default:
		return -1
	}
}

func TestVerificationKeyUnionDegenerate(t *testing.T) {
	set := VerificationKeySet{
		Keys: []VerificationKey{
			[]uint8{1, 2, 3},
			123,
			"abc",
		},
	}

	if got := checkVerificationKey(set); got != 3 {
		t.Fatalf("checkVerificationKey(VerificationKeySet) = %d, want 3", got)
	}

	if got := checkVerificationKey([]uint8{1, 2}); got != 100 {
		t.Fatalf("checkVerificationKey([]uint8) = %d, want 100", got)
	}

	if got := checkVerificationKey(123); got != 100 {
		t.Fatalf("checkVerificationKey(int) = %d, want 100", got)
	}

	if got := checkVerificationKey("hello"); got != 100 {
		t.Fatalf("checkVerificationKey(string) = %d, want 100", got)
	}
}
