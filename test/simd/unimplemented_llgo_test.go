//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"strings"
	"testing"
)

type unsupportedCall struct {
	name string
	call func()
}

func TestUnimplementedSIMD(t *testing.T) {
	var x archsimd.Uint32x4
	symbol, calls := unsupportedCalls(x)
	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				err, ok := recover().(interface{ Error() string })
				if !ok || !strings.HasPrefix(err.Error(), "runtime error: unimplemented SIMD intrinsic: "+symbol) {
					t.Fatalf("unexpected panic: %v", err)
				}
			}()
			tc.call()
			t.Fatal("unimplemented intrinsic returned")
		})
	}
	// Go helper bodies remain executable; the fallback applies to declarations.
	if x.Len() != 4 {
		t.Fatal("Go helper was replaced by the fallback")
	}
}

//go:noinline
func invokeUnsupportedPair[A, R any](f func(A, A) R, x A) R { return f(x, x) }
