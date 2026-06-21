package gotest

import (
	"runtime"
	"strings"
	"testing"
)

var (
	runtimeErrorIntSink     int
	runtimeErrorAnySink     any
	runtimeErrorArrayPtr    *[10]int
	runtimeErrorBigArrayPtr *[10000]int
)

func TestRecoverRuntimeErrorClassification(t *testing.T) {
	var zero int
	var zero64 int64
	var index = 99999
	arrayPtr := new([10]int)
	var slice []int
	var iface any = 1

	tests := []struct {
		name string
		want string
		f    func()
	}{
		{
			name: "int-div-zero",
			want: "integer divide by zero",
			f: func() {
				runtimeErrorIntSink = 1 / zero
			},
		},
		{
			name: "int64-div-zero",
			want: "integer divide by zero",
			f: func() {
				runtimeErrorIntSink = int(1 / zero64)
			},
		},
		{
			name: "nil-array-pointer-index-zero",
			want: "nil pointer dereference",
			f: func() {
				runtimeErrorIntSink = runtimeErrorArrayPtr[0]
			},
		},
		{
			name: "nil-array-pointer-index-one",
			want: "nil pointer dereference",
			f: func() {
				runtimeErrorIntSink = runtimeErrorArrayPtr[1]
			},
		},
		{
			name: "nil-array-pointer-index-large",
			want: "nil pointer dereference",
			f: func() {
				runtimeErrorIntSink = runtimeErrorBigArrayPtr[5000]
			},
		},
		{
			name: "array-bounds",
			want: "index out of range",
			f: func() {
				runtimeErrorIntSink = arrayPtr[index]
			},
		},
		{
			name: "slice-bounds",
			want: "index out of range",
			f: func() {
				runtimeErrorIntSink = slice[index]
			},
		},
		{
			name: "type-concrete",
			want: "int, not string",
			f: func() {
				runtimeErrorAnySink = iface.(string)
			},
		},
		{
			name: "type-interface",
			want: "missing method runtimeErrorMissingMethod",
			f: func() {
				runtimeErrorAnySink = iface.(runtimeErrorMissingMethod)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectRecoverRuntimeError(t, tt.want, tt.f)
		})
	}
}

func expectRecoverRuntimeError(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected runtime panic containing %q", want)
		}
		runtimeErr, ok := err.(runtime.Error)
		if !ok {
			t.Fatalf("panic type = %T, want runtime.Error", err)
		}
		if got := runtimeErr.Error(); !strings.Contains(got, want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}()
	f()
}

type runtimeErrorMissingMethod interface {
	runtimeErrorMissingMethod()
}
