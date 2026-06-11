package gotest

import (
	"runtime"
	"strings"
	"testing"
)

type runtimeErrorMissingMethod interface {
	runtimeErrorMissingMethod()
}

var (
	runtimeErrorSink        any
	runtimeErrorIntSink     int
	runtimeErrorArrayPtr    *[10]int
	runtimeErrorBigArrayPtr *[10000]int
)

func TestRecoveredRuntimePanicsAreErrors(t *testing.T) {
	var index = 99999
	arrayPtr := new([10]int)

	tests := []struct {
		name string
		want []string
		f    func()
	}{
		{
			name: "index",
			want: []string{"runtime error:", "index out of range"},
			f: func() {
				s := []byte{1}
				i := 2
				runtimeErrorSink = s[i]
			},
		},
		{
			name: "array bounds",
			want: []string{"runtime error:", "index out of range"},
			f: func() {
				runtimeErrorIntSink = arrayPtr[index]
			},
		},
		{
			name: "slice",
			want: []string{"runtime error:", "slice bounds out of range"},
			f: func() {
				s := []byte{1}
				hi := 2
				runtimeErrorSink = s[:hi]
			},
		},
		{
			name: "divide",
			want: []string{"runtime error:", "integer divide by zero"},
			f: func() {
				z := 0
				runtimeErrorSink = 1 / z
			},
		},
		{
			name: "nil dereference",
			want: []string{"runtime error:", "nil pointer dereference"},
			f: func() {
				var p *int
				runtimeErrorSink = *p
			},
		},
		{
			name: "nil array pointer index zero",
			want: []string{"runtime error:", "nil pointer dereference"},
			f: func() {
				runtimeErrorIntSink = runtimeErrorArrayPtr[0]
			},
		},
		{
			name: "nil array pointer index one",
			want: []string{"runtime error:", "nil pointer dereference"},
			f: func() {
				runtimeErrorIntSink = runtimeErrorArrayPtr[1]
			},
		},
		{
			name: "nil array pointer index large",
			want: []string{"runtime error:", "nil pointer dereference"},
			f: func() {
				runtimeErrorIntSink = runtimeErrorBigArrayPtr[5000]
			},
		},
		{
			name: "slice to array",
			want: []string{"runtime error:", "cannot convert slice with length 1 to array or pointer to array with length 2"},
			f: func() {
				s := []byte{1}
				runtimeErrorSink = [2]byte(s)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := recoverRuntimeErrorValue(t, tt.f)
			assertRuntimeErrorContains(t, err, tt.want...)
		})
	}
}

func TestRecoveredTypeAssertionPanicsAreRuntimeErrors(t *testing.T) {
	t.Run("concrete", func(t *testing.T) {
		var v any = 1
		err := recoverRuntimeErrorValue(t, func() {
			runtimeErrorSink = v.(string)
		})
		assertRuntimeErrorContains(t, err, "interface conversion", "int", "not string")
	})

	t.Run("nil interface", func(t *testing.T) {
		var v any
		err := recoverRuntimeErrorValue(t, func() {
			runtimeErrorSink = v.(string)
		})
		assertRuntimeErrorContains(t, err, "interface conversion", "is nil", "not string")
	})

	t.Run("missing method", func(t *testing.T) {
		var v any = 1
		err := recoverRuntimeErrorValue(t, func() {
			runtimeErrorSink = v.(runtimeErrorMissingMethod)
		})
		assertRuntimeErrorContains(t, err, "interface conversion", "int is not", "missing method runtimeErrorMissingMethod")
	})
}

func recoverRuntimeErrorValue(t *testing.T, f func()) runtime.Error {
	t.Helper()
	var rec any
	func() {
		defer func() {
			rec = recover()
		}()
		f()
	}()
	if rec == nil {
		t.Fatal("expected panic")
	}
	err := rec.(error)
	rerr := rec.(runtime.Error)
	if err.Error() != rerr.Error() {
		t.Fatalf("error text mismatch: error=%q runtime.Error=%q", err.Error(), rerr.Error())
	}
	return rerr
}

func assertRuntimeErrorContains(t *testing.T, err error, wants ...string) {
	t.Helper()
	got := err.Error()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}
}
