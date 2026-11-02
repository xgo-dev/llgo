//go:build llgo && (darwin || linux) && !wasm && !baremetal

package abicall

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestPackedCallGCNestedAndUnwind(t *testing.T) {
	for _, depth := range []int64{0, 3} {
		payload := []byte{42}
		b := block{a: 20, b: 22, depth: depth, payload: unsafe.Pointer(&payload[0])}
		got := libcCall(trampoline(), unsafe.Pointer(&b))
		saved := currentErrno()
		if got != 18 || b.result != 42 || b.traced != 1 {
			t.Fatalf("call=%d result=%d unwind=%d", got, b.result, b.traced)
		}
		if saved != errnoRange() {
			t.Fatalf("errno=%d", saved)
		}
	}
}

func TestPackedCallSignal(t *testing.T) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	defer signal.Stop(signals)
	payload := []byte{42}
	b := block{a: 20, b: 22, signal: 1, payload: unsafe.Pointer(&payload[0])}
	if libcCall(trampoline(), unsafe.Pointer(&b)) != 18 {
		t.Fatal("call result")
	}
	select {
	case <-signals:
	case <-time.After(2 * time.Second):
		t.Fatal("signal lost across packed call")
	}
}

func TestPackedCallNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil trampoline did not panic")
		}
	}()
	libcCall(nil, nil)
}
