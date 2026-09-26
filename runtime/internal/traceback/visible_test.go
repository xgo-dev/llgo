package traceback

import "testing"

func TestVisible(t *testing.T) {
	for _, tc := range []struct {
		name                string
		system, first, want bool
	}{
		{"", true, true, true}, {"", false, true, false},
		{"main.main", false, true, true}, {"pthread_start", false, true, false},
		{"runtime.Callers", false, true, true}, {"runtime.", false, true, false},
		{"runtime.goexit", false, true, false}, {"runtime.goexit", true, true, true},
		{"runtime.(*Func).Entry", false, true, true}, {"runtime.Func.Name", false, true, true},
		{"runtime.(*funcInfo).Entry", false, true, false}, {"runtime.Func.name", false, true, false},
		{"runtime.(*Func).", false, true, false},
		{"runtime.gopanic", false, true, false}, {"runtime.gopanic", false, false, true},
		{"runtime.runFinalizers", false, true, true}, {"runtime.runCleanups", false, true, true},
		{"github.com/xgo-dev/llgo/runtime/internal/runtime.Panic", false, true, false},
		{"github.com/xgo-dev/llgo/runtime/internal/runtime.Panic", false, false, true},
		{"github.com/xgo-dev/llgo/runtime/internal/runtime.ChanRecv", false, false, false},
	} {
		if got := Visible(tc.name, tc.system, tc.first); got != tc.want {
			t.Errorf("Visible(%q,%v,%v)=%v, want %v", tc.name, tc.system, tc.first, got, tc.want)
		}
	}
}
