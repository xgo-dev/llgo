package wasmtest

import (
	"crypto/sha256"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStandardLibraryWasmAssembly(t *testing.T) {
	if got := math.Floor(3.75); got != 3 {
		t.Fatalf("math.Floor(3.75) = %v, want 3", got)
	}
	const wantSHA256 = "336154bf67f765f8f75d16a0accee61b5ee5f6a75b2a2905703df913bd550f3e"
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte("wasm"))); got != wantSHA256 {
		t.Fatalf("sha256.Sum256(wasm) = %s, want %s", got, wantSHA256)
	}
}

func TestScheduler(t *testing.T) {
	done := make(chan int, 1)
	go func() {
		done <- 42
	}()

	select {
	case got := <-done:
		if got != 42 {
			t.Fatalf("goroutine result = %d, want 42", got)
		}
	case <-time.After(time.Second):
		t.Fatal("goroutine did not make progress")
	}
}

//go:noinline
func functionValueSymbolizationTarget() {}

func TestFunctionValueSymbolization(t *testing.T) {
	pc := reflect.ValueOf(functionValueSymbolizationTarget).Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil || !strings.HasSuffix(fn.Name(), ".functionValueSymbolizationTarget") {
		t.Fatalf("FuncForPC(reflect.Value.Pointer()) = %v for %#x", fn, pc)
	}
}

func TestPanicRecoverAndCaller(t *testing.T) {
	defer func() {
		got := recover()
		if got != "wasm-test-panic" {
			t.Fatalf("recover = %v, want wasm-test-panic", got)
		}
	}()

	if _, file, line, ok := runtime.Caller(0); !ok || file == "" || line == 0 {
		t.Fatalf("runtime.Caller = %q:%d, %v", file, line, ok)
	}
	panic("wasm-test-panic")
}
