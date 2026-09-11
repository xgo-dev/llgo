package gotest

import (
	"os"
	"runtime"
	"strings"
)

const (
	finalizerInvalidCaseEnv       = "LLGO_TEST_FINALIZER_INVALID_CASE"
	finalizerInvalidCaseArgPrefix = "-llgo.finalizer-invalid-case="
)

var finalizerInvalidCases = []string{"non-function", "no parameters", "two parameters", "variadic", "wrong type"}

func init() {
	name := os.Getenv(finalizerInvalidCaseEnv)
	// Wasm guests cannot create a subprocess, and WASI does not inherit an
	// arbitrary host environment. The host acceptance driver reuses the test
	// binary and selects the same failure path through argv instead.
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, finalizerInvalidCaseArgPrefix) {
			name = strings.TrimPrefix(arg, finalizerInvalidCaseArgPrefix)
		}
	}
	if name == "" {
		return
	}

	p := &finalizerAssignableValue{keep: new(int)}
	switch name {
	case "non-function":
		runtime.SetFinalizer(p, 1)
	case "no parameters":
		runtime.SetFinalizer(p, func() {})
	case "two parameters":
		runtime.SetFinalizer(p, func(*finalizerAssignableValue, int) {})
	case "variadic":
		runtime.SetFinalizer(p, func(...*finalizerAssignableValue) {})
	case "wrong type":
		runtime.SetFinalizer(p, func(*int) {})
	default:
		panic("unknown invalid finalizer case: " + name)
	}
	// A conforming runtime terminates in every case above. Make acceptance fail
	// unambiguously if it returns instead.
	os.Exit(0)
}
