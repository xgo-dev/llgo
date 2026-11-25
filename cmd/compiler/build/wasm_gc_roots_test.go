package build

import (
	"github.com/xgo-dev/llgo/cmd/compiler/wasmworkers"
	"testing"
)

func TestThreadLocalGCRootSelection(t *testing.T) {
	for _, tc := range []struct {
		name, goos, threads string
		gc                  bool
		workers             int
		want                bool
	}{
		{"WASI GC", "wasip1", "1", true, 1, true},
		{"WASI nogc", "wasip1", "1", false, 1, false},
		{"browser workers", "js", "", true, 2, true},
		{"browser nogc", "js", "", false, 2, false},
		{"browser single worker", "js", "", true, 1, false},
		{"native", "linux", "1", false, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LLGO_WASI_THREADS", tc.threads)
			got := useThreadLocalGCRoots(&Config{Goos: tc.goos}, tc.gc, wasmworkers.Config{Count: tc.workers})
			if got != tc.want {
				t.Fatalf("thread-local roots = %v, want %v", got, tc.want)
			}
		})
	}
}
