package gotest

import "testing"

type nilAssertInterface interface {
	M()
}

type nilAssertImpl struct{}

func (*nilAssertImpl) M() {}

// Keep nil interface construction behind a function so the assertion is
// generated at runtime instead of folded from a local nil variable.
func nilAssertValue(ok bool) nilAssertInterface {
	if ok {
		return &nilAssertImpl{}
	}
	return nil
}

func TestNilInterfaceSameTypeAssert(t *testing.T) {
	x := nilAssertValue(false)
	if _, ok := x.(nilAssertInterface); ok {
		t.Fatal("nil interface asserted to the same interface type")
	}

	x = nilAssertValue(true)
	if _, ok := x.(nilAssertInterface); !ok {
		t.Fatal("non-nil interface rejected by same interface assertion")
	}

	x = (*nilAssertImpl)(nil)
	if _, ok := x.(nilAssertInterface); !ok {
		t.Fatal("typed nil interface rejected by same interface assertion")
	}
}

func TestNilInterfaceSameTypeAssertPanics(t *testing.T) {
	x := nilAssertValue(false)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for nil interface same-type assert")
		}
	}()
	_ = x.(nilAssertInterface)
}
