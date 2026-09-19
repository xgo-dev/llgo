package gotest

import (
	"reflect"
	"testing"
)

func genericDeferSequence(yield func(int) bool) {
	for i := 1; i <= 3; i++ {
		if !yield(i) {
			return
		}
	}
}

func genericRangeDefer[T ~string](tag T, events *[]string) {
	for value := range genericDeferSequence {
		defer func() { *events = append(*events, string(tag)) }()
		defer func() { *events = append(*events, string(rune('0'+value))) }()
	}
}

type genericDeferSet[T comparable] struct{ values []T }

func (s *genericDeferSet[T]) keys() func(func(T) bool) {
	return func(yield func(T) bool) {
		for _, value := range s.values {
			if !yield(value) {
				return
			}
		}
	}
}

func (s *genericDeferSet[T]) visit(events *[]T) {
	for value := range s.keys() {
		defer func() { *events = append(*events, value) }()
	}
}

func TestGenericRangeFuncDefers(t *testing.T) {
	var functionEvents []string
	genericRangeDefer("f", &functionEvents)
	if want := []string{"3", "f", "2", "f", "1", "f"}; !reflect.DeepEqual(functionEvents, want) {
		t.Errorf("generic function defers = %v, want %v", functionEvents, want)
	}

	var methodEvents []int
	(&genericDeferSet[int]{[]int{1, 2, 3}}).visit(&methodEvents)
	if want := []int{3, 2, 1}; !reflect.DeepEqual(methodEvents, want) {
		t.Errorf("generic method defers = %v, want %v", methodEvents, want)
	}
}
