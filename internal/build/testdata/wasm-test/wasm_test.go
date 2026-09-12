package wasmtest

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"runtime"
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

func TestSyncFileIO(t *testing.T) {
	path := "llgo-fs-sync.txt"
	const want = "hello-sync\n"
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ReadFile = %q, want %q", got, want)
	}
	if _, err := fmt.Println("hello-println"); err != nil {
		t.Fatalf("fmt.Println: %v", err)
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
