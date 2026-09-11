//go:build wasm

package signal_test

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"testing"
)

// Go's wasm runtime has no OS signal delivery. Registration and cancellation
// still have public contracts; do not send a signal to the WASI guest itself,
// where even signal zero is a request to terminate the process.
func TestWasmNotifyRegistration(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Interrupt)
	signal.Stop(c)
	signal.Stop(c)
	signal.Ignore(os.Interrupt)
	// Do not query an OS disposition: Go's wasm signal_ignored indexes its
	// zero-length (_NSIG=0) signal table. An unknown Signal is rejected by
	// os/signal itself and has a portable negative query result.
	if signal.Ignored(unknownWasmSignal{}) {
		t.Fatal("unknown signal reported as ignored")
	}
	signal.Reset(os.Interrupt)
	select {
	case sig := <-c:
		t.Fatalf("registration synthesized signal %v", sig)
	default:
	}
}

type unknownWasmSignal struct{}

func (unknownWasmSignal) Signal()        {}
func (unknownWasmSignal) String() string { return "unknown wasm signal" }

func TestWasmNotifyRejectsNilChannel(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Notify(nil) did not panic")
		}
	}()
	signal.Notify(nil, os.Interrupt)
}

func TestWasmNotifyContextCancellation(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		parent, cancel := context.WithCancel(context.Background())
		ctx, stop := signal.NotifyContext(parent, os.Interrupt)
		if ctx.Err() != nil {
			t.Fatalf("new context is already canceled: %v", ctx.Err())
		}
		if cancelParent {
			cancel()
		} else {
			stop()
		}
		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("context error = %v", ctx.Err())
		}
		stop()
		cancel()
	}
}
