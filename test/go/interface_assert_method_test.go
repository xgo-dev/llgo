package gotest

import (
	"testing"

	"github.com/xgo-dev/llgo/test/go/testdata/assertmethods"
)

type assertMethodBase interface{ A() int }
type assertMethodExtended interface {
	A() int
	B() int
}

type assertMethodPair struct{ value int }

func (p *assertMethodPair) A() int { return 1 }
func (p *assertMethodPair) B() int {
	if p == nil {
		return -1
	}
	return p.value
}

type assertMethodMany struct{ assertMethodPair }

func (*assertMethodMany) C() {}
func (*assertMethodMany) D() {}
func (*assertMethodMany) E() {}
func (*assertMethodMany) F() {}
func (*assertMethodMany) G() {}
func (*assertMethodMany) H() {}

type assertMethodMissing struct{}

func (*assertMethodMissing) A() int { return 1 }

type assertMethodWrongSignature struct{}

func (*assertMethodWrongSignature) A() int    { return 1 }
func (*assertMethodWrongSignature) B() string { return "wrong" }

//go:noinline
func checkAssertMethods(v assertMethodBase) (int, bool) {
	w, ok := v.(assertMethodExtended)
	if !ok {
		return 0, false
	}
	return w.B(), true
}

func TestInterfaceAssertMethodTable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value assertMethodBase
		want  int
		ok    bool
	}{
		{"two-methods", &assertMethodPair{7}, 7, true},
		{"eight-methods", &assertMethodMany{assertMethodPair{9}}, 9, true},
		{"missing-method", &assertMethodMissing{}, 0, false},
		{"wrong-signature", &assertMethodWrongSignature{}, 0, false},
		{"typed-nil", (*assertMethodPair)(nil), -1, true},
		{"nil-interface", nil, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				if got, ok := checkAssertMethods(tc.value); got != tc.want || ok != tc.ok {
					t.Fatalf("assertion = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.ok)
				}
			}
		})
	}
}

type assertMethodPrivate struct{}

func (assertMethodPrivate) private() int { return 23 }

//go:noinline
func assertLocalPrivate(v any) (int, bool) {
	w, ok := v.(interface{ private() int })
	if !ok {
		return 0, false
	}
	return w.private(), true
}

func TestInterfaceAssertMethodPackage(t *testing.T) {
	local, foreign := any(assertMethodPrivate{}), any(assertmethods.Value{})
	if got, ok := assertLocalPrivate(local); !ok || got != 23 {
		t.Fatalf("local method call = (%d, %v), want (23, true)", got, ok)
	}
	if _, ok := assertLocalPrivate(foreign); ok {
		t.Fatal("local interface ignored the unexported method's package")
	}
	if _, ok := assertmethods.Check(local); ok {
		t.Fatal("foreign interface ignored the unexported method's package")
	}
	if got, ok := assertmethods.Check(foreign); !ok || got != 17 {
		t.Fatalf("foreign method call = (%d, %v), want (17, true)", got, ok)
	}
}
