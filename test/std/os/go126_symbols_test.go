//go:build go1.26

package os_test

import (
	"errors"
	"os"
	"testing"
)

func TestProcessWithHandle(t *testing.T) {
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	called := false
	err = process.WithHandle(func(handle uintptr) {
		called = true
		if handle == 0 {
			t.Error("WithHandle supplied a zero handle")
		}
	})
	if errors.Is(err, os.ErrNoHandle) {
		if called {
			t.Fatal("WithHandle called its callback while returning ErrNoHandle")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("WithHandle did not call its callback")
	}
}
