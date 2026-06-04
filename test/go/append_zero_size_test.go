package gotest

import (
	"fmt"
	"strings"
	"testing"
)

type appendZeroSize struct{}

func TestAppendZeroSizedElementsUpdatesSliceHeader(t *testing.T) {
	got := append([]appendZeroSize{}, make([]appendZeroSize, 2)...)
	if len(got) != 2 {
		t.Fatalf("len(append(empty, make(2)...)) = %d, want 2", len(got))
	}
	if cap(got) < len(got) {
		t.Fatalf("cap after zero-sized append = %d, want at least len %d", cap(got), len(got))
	}

	got = append(got, appendZeroSize{})
	if len(got) != 3 {
		t.Fatalf("len after single zero-sized append = %d, want 3", len(got))
	}
	if cap(got) < len(got) {
		t.Fatalf("cap after single zero-sized append = %d, want at least len %d", cap(got), len(got))
	}
}

func TestAppendZeroSizedElementsWithinCapacity(t *testing.T) {
	base := make([]appendZeroSize, 1, 4)
	got := append(base, make([]appendZeroSize, 2)...)
	if len(got) != 3 {
		t.Fatalf("len(append(len=1 cap=4, make(2)...)) = %d, want 3", len(got))
	}
	if cap(got) != 4 {
		t.Fatalf("cap after append within capacity = %d, want 4", cap(got))
	}
}

func TestAppendZeroSizedElementsLengthOverflow(t *testing.T) {
	const maxInt = int(^uint(0) >> 1)

	s := make([]appendZeroSize, maxInt)
	expectAppendPanicContaining(t, "len out of range", func() {
		s = append(s, appendZeroSize{})
	})

	oneElem := make([]appendZeroSize, 1)
	expectAppendPanicContaining(t, "len out of range", func() {
		s = append(s, oneElem...)
	})

	a := make([]appendZeroSize, maxInt)
	b := make([]appendZeroSize, maxInt)
	expectAppendPanicContaining(t, "len out of range", func() {
		_ = append(a, b...)
	})
}

func expectAppendPanicContaining(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if got := appendPanicString(err); !strings.Contains(got, want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}()
	f()
}

func appendPanicString(v any) string {
	if err, ok := v.(interface{ Error() string }); ok {
		return err.Error()
	}
	return fmt.Sprint(v)
}
