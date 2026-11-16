package gotest

import "testing"

type branchDeferStep func([]uint64) error

var branchDeferCallback any
var branchDeferCleanup uint64

//go:noinline
func branchDeferInvoke(words []uint64) error {
	if words[0] == 0 {
		panic("branch callback")
	}
	words[0] = words[0]*17 + 5
	return nil
}

//go:noinline
func branchDeferCapture(words []uint64, withSlice bool) error {
	f, ok := branchDeferCallback.(branchDeferStep)
	if !ok {
		panic("callback type")
	}
	var temporary []uint64
	if withSlice {
		temporary = []uint64{23, 41}
	}
	panicked := true
	defer func() {
		if panicked {
			branchDeferCleanup++
		}
		for _, x := range temporary {
			branchDeferCleanup += x
		}
	}()
	err := f(words)
	panicked = false
	return err
}

//go:noinline
func branchDeferRecover(words []uint64, withSlice bool) (value any) {
	defer func() { value = recover() }()
	branchDeferCapture(words, withSlice)
	return nil
}

func TestDeferCaptureAfterTypeAssertion(t *testing.T) {
	previous := branchDeferCallback
	defer func() { branchDeferCallback = previous }()
	branchDeferCallback = branchDeferStep(branchDeferInvoke)
	for _, withSlice := range []bool{false, true} {
		branchDeferCleanup = 0
		words := []uint64{7}
		if err := branchDeferCapture(words, withSlice); err != nil || words[0] != 124 {
			t.Fatalf("callback result: words=%v err=%v", words, err)
		}
		want := uint64(0)
		if withSlice {
			want = 64
		}
		if branchDeferCleanup != want {
			t.Fatalf("return cleanup=%d, want %d", branchDeferCleanup, want)
		}
		words[0] = 0
		if got := branchDeferRecover(words, withSlice); got != "branch callback" {
			t.Fatalf("recover=%v", got)
		}
		if branchDeferCleanup != 2*want+1 {
			t.Fatalf("panic cleanup=%d, want %d", branchDeferCleanup, 2*want+1)
		}
	}
	branchDeferCallback = 1
	if got := branchDeferRecover([]uint64{7}, false); got != "callback type" {
		t.Fatalf("invalid callback recover=%v", got)
	}
}
