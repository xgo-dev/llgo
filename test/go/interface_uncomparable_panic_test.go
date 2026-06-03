package gotest

import (
	"fmt"
	"strings"
	"testing"
)

func TestInterfaceCompareUncomparableDirectValuePanics(t *testing.T) {
	type uncomparableDirect struct {
		f func()
	}
	v := uncomparableDirect{f: func() {}}
	expectPanicContaining(t, "comparing uncomparable type gotest.uncomparableDirect", func() {
		_ = any(v) == any(v)
	})
}

func expectPanicContaining(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if _, ok := err.(interface{ RuntimeError() }); !ok {
			t.Fatalf("panic type = %T, want runtime.Error", err)
		}
		if got := fmt.Sprint(err); !strings.Contains(got, want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}()
	f()
}
