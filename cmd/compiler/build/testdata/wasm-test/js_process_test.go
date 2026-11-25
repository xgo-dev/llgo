//go:build js && wasm && !llgo.wasm.emscripten

package wasmtest

import (
	"errors"
	"syscall"
	"syscall/js"
	"testing"
)

func TestGoJSBrowserProcessFallback(t *testing.T) {
	process := js.Global().Get("process")
	if process.IsUndefined() {
		t.Fatal("browser host did not install Go's process fallback")
	}
	for name, got := range map[string]int{
		"getuid":  syscall.Getuid(),
		"getgid":  syscall.Getgid(),
		"geteuid": syscall.Geteuid(),
		"getegid": syscall.Getegid(),
		"pid":     syscall.Getpid(),
		"ppid":    syscall.Getppid(),
	} {
		if got != -1 {
			t.Errorf("%s = %d, want -1", name, got)
		}
	}
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGCHLD); !errors.Is(err, syscall.ENOSYS) {
		t.Fatalf("Kill = %v, want ENOSYS", err)
	}
	if groups, err := syscall.Getgroups(); groups != nil || !errors.Is(err, syscall.ENOSYS) {
		t.Fatalf("Getgroups = %v, %v; want nil, ENOSYS", groups, err)
	}
}
