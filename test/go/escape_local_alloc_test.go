//go:build !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
)

func TestDeferCaptureBranchAllocations(t *testing.T) {
	previous := branchDeferCallback
	defer func() { branchDeferCallback = previous }()
	branchDeferCallback = branchDeferStep(branchDeferInvoke)
	words := []uint64{7}
	// Warm metadata and defer machinery before measuring the retained flag.
	for i := 0; i < 64; i++ {
		branchDeferCapture(words, false)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 10000; i++ {
		branchDeferCapture(words, false)
	}
	runtime.ReadMemStats(&after)
	// Process-wide counters can include unrelated runtime activity. A retained
	// capture allocates once per call; allow only incidental allocations.
	if n := after.Mallocs - before.Mallocs; n > 16 {
		t.Fatalf("10000 frame-bounded defer calls allocated %d objects", n)
	}
}

func TestUnsafeLocalViewsAllocations(t *testing.T) {
	for i := range 64 {
		unsafeLocalViews(byte(i))
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	var sum int
	for i := range 10000 {
		sum += unsafeLocalViews(byte(i % 128))
	}
	runtime.ReadMemStats(&after)
	if sum == 0 {
		t.Fatal("missing view result")
	}
	if n := after.Mallocs - before.Mallocs; n > 16 {
		t.Fatalf("10000 local unsafe views allocated %d objects", n)
	}
}

type forwardedCallbackCaller struct {
	active bool
	owner  *forwardedCallbackHost
}

type forwardedCallbackHost struct {
	calls uint64
	read  func(*forwardedCallbackCaller) ([]byte, bool, error)
}

func (h *forwardedCallbackHost) expire(c *forwardedCallbackCaller) {
	c.active = false
	h.calls++
}

func (h *forwardedCallbackHost) direct(fn func(*forwardedCallbackCaller) (*[]byte, error)) (*[]byte, error) {
	c := &forwardedCallbackCaller{active: true, owner: h}
	defer h.expire(c)
	return fn(c)
}

func (h *forwardedCallbackHost) wrapped(fn func(*forwardedCallbackCaller) (*[]byte, error)) (*[]byte, error) {
	return h.invoke(&forwardedCallbackCaller{active: true, owner: h}, fn)
}

func (h *forwardedCallbackHost) invoke(c *forwardedCallbackCaller, fn func(*forwardedCallbackCaller) (*[]byte, error)) (*[]byte, error) {
	defer h.expire(c)
	return fn(c)
}

//go:noinline
func forwardedCallbackDirectRead(h *forwardedCallbackHost) int {
	var result []byte
	var ready bool
	_, err := h.direct(func(c *forwardedCallbackCaller) (*[]byte, error) {
		var err error
		result, ready, err = h.read(c)
		return nil, err
	})
	if err != nil || !ready {
		panic("unexpected direct result")
	}
	return len(result)
}

//go:noinline
func forwardedCallbackWrappedRead(h *forwardedCallbackHost) int {
	var result []byte
	var ready bool
	_, err := h.wrapped(func(c *forwardedCallbackCaller) (*[]byte, error) {
		var err error
		result, ready, err = h.read(c)
		return nil, err
	})
	if err != nil || !ready {
		panic("unexpected wrapped result")
	}
	return len(result)
}

var forwardedCallbackPayload = [1]byte{7}

func TestForwardedCallbackCaptureAllocations(t *testing.T) {
	h := &forwardedCallbackHost{
		read: func(c *forwardedCallbackCaller) ([]byte, bool, error) {
			if !c.active || c.owner == nil {
				panic("expired caller")
			}
			return forwardedCallbackPayload[:], true, nil
		},
	}
	for range 64 {
		if forwardedCallbackDirectRead(h) != 1 || forwardedCallbackWrappedRead(h) != 1 {
			t.Fatal("unexpected callback read length")
		}
	}
	const calls = 4096
	measure := func(fn func(*forwardedCallbackHost) int) uint64 {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range calls {
			if fn(h) != 1 {
				t.Fatal("unexpected callback read length")
			}
		}
		runtime.ReadMemStats(&after)
		return after.Mallocs - before.Mallocs
	}
	directAllocs := measure(forwardedCallbackDirectRead)
	wrappedAllocs := measure(forwardedCallbackWrappedRead)
	// Both variants allocate one caller record per invocation; neither path
	// may additionally heap-allocate the callback's captured result locals.
	const maxAllocs = calls + 64
	if directAllocs > maxAllocs || wrappedAllocs > maxAllocs {
		t.Fatalf("callback captures heap-allocated: direct=%d wrapped=%d (max %d for %d calls)", directAllocs, wrappedAllocs, maxAllocs, calls)
	}
	if wrappedAllocs > directAllocs+64 {
		t.Fatalf("forwarded callback allocated %d objects vs %d direct for %d calls", wrappedAllocs, directAllocs, calls)
	}
}
