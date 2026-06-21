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

func TestComparableTypeParamInterfaceValuePanics(t *testing.T) {
	type uncomparableParam struct {
		s []int
	}
	v := any(uncomparableParam{s: []int{1}})
	expectPanicContaining(t, "comparing uncomparable type gotest.uncomparableParam", func() {
		_ = eqComparable[any](v, v)
	})
}

func eqComparable[T comparable](a, b T) bool {
	return a == b
}

func TestInterfaceCompareEvaluationPanicsAreRuntimeErrors(t *testing.T) {
	var (
		x interface{}
		p *int
		s []int
		l *interface{}
		r []*int
	)
	tests := []struct {
		name string
		want string
		f    func()
	}{
		{"switch case type assertion", "", func() {
			switch x {
			case x.(*int):
			}
		}},
		{"interface conversion", "", func() { _ = x == x.(error) }},
		{"type assertion", "", func() { _ = x == x.(*int) }},
		{"out of bounds", "index out of range", func() { _ = x == s[1] }},
		{"nil pointer dereference #1", "invalid memory address", func() { _ = x == *p }},
		{"nil pointer dereference #2", "", func() { _ = *l == r[0] }},
		{"nil pointer dereference #3", "", func() { _ = *l == any(r[0]) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectPanicContaining(t, tt.want, tt.f)
		})
	}
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
