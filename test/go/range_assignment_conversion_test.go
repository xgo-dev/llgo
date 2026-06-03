package gotest

import (
	"reflect"
	"testing"
)

type rangeAssignmentIface interface{ rangeAssignmentM() int }

type rangeAssignmentKey int
type rangeAssignmentValue int
type rangeAssignmentSlice []int

func (rangeAssignmentKey) rangeAssignmentM() int   { return 0 }
func (rangeAssignmentValue) rangeAssignmentM() int { return 0 }

func TestRangeAssignmentConversions(t *testing.T) {
	k, v := rangeArrayAny[rangeAssignmentValue]()
	rangeAssignmentMatch2(t, "int", "rangeAssignmentValue", k, v)
	k, v = rangeArrayIface[rangeAssignmentValue]()
	rangeAssignmentMatch2(t, "int", "rangeAssignmentValue", k, v)
	rangeAssignmentMatch1(t, "rangeAssignmentValue", rangeChanAny[rangeAssignmentValue]())
	rangeAssignmentMatch1(t, "rangeAssignmentValue", rangeChanIface[rangeAssignmentValue]())
	k, v = rangeMapAny[rangeAssignmentKey, rangeAssignmentValue]()
	rangeAssignmentMatch2(t, "rangeAssignmentKey", "rangeAssignmentValue", k, v)
	k, v = rangeMapIface[rangeAssignmentKey, rangeAssignmentValue]()
	rangeAssignmentMatch2(t, "rangeAssignmentKey", "rangeAssignmentValue", k, v)
	k, v = rangeSliceAny[rangeAssignmentValue]()
	rangeAssignmentMatch2(t, "int", "rangeAssignmentValue", k, v)
	k, v = rangeSliceIface[rangeAssignmentValue]()
	rangeAssignmentMatch2(t, "int", "rangeAssignmentValue", k, v)
	if got := rangeSliceChangeType(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("range assignment ChangeType result = %v, want [1 2]", got)
	}
}

func rangeAssignmentMatch1(t *testing.T, want string, arg any) {
	t.Helper()
	if got := reflect.TypeOf(arg).Name(); got != want {
		t.Fatalf("arg type = %q, want %q", got, want)
	}
}

func rangeAssignmentMatch2(t *testing.T, want0, want1 string, arg0, arg1 any) {
	t.Helper()
	if got := reflect.TypeOf(arg0).Name(); got != want0 {
		t.Fatalf("arg 0 type = %q, want %q", got, want0)
	}
	if got := reflect.TypeOf(arg1).Name(); got != want1 {
		t.Fatalf("arg 1 type = %q, want %q", got, want1)
	}
}

func rangeArrayAny[V any]() (k, v any) {
	for k, v = range [...]V{rangeAssignmentZero[V]()} {
	}
	return
}

func rangeArrayIface[V rangeAssignmentIface]() (k any, v rangeAssignmentIface) {
	for k, v = range [...]V{rangeAssignmentZero[V]()} {
	}
	return
}

func rangeChanAny[V any]() (v any) {
	for v = range rangeAssignmentChanOf(rangeAssignmentZero[V]()) {
	}
	return
}

func rangeChanIface[V rangeAssignmentIface]() (v rangeAssignmentIface) {
	for v = range rangeAssignmentChanOf(rangeAssignmentZero[V]()) {
	}
	return
}

func rangeMapAny[K comparable, V any]() (k, v any) {
	for k, v = range map[K]V{rangeAssignmentZero[K](): rangeAssignmentZero[V]()} {
	}
	return
}

func rangeMapIface[K interface {
	rangeAssignmentIface
	comparable
}, V rangeAssignmentIface]() (k, v rangeAssignmentIface) {
	for k, v = range map[K]V{rangeAssignmentZero[K](): rangeAssignmentZero[V]()} {
	}
	return
}

func rangeSliceAny[V any]() (k, v any) {
	for k, v = range []V{rangeAssignmentZero[V]()} {
	}
	return
}

func rangeSliceIface[V rangeAssignmentIface]() (k any, v rangeAssignmentIface) {
	for k, v = range []V{rangeAssignmentZero[V]()} {
	}
	return
}

func rangeSliceChangeType() (v []int) {
	for _, v = range []rangeAssignmentSlice{{1, 2}} {
	}
	return
}

func rangeAssignmentChanOf[T any](elems ...T) chan T {
	c := make(chan T, len(elems))
	for _, elem := range elems {
		c <- elem
	}
	close(c)
	return c
}

func rangeAssignmentZero[T any]() (_ T) { return }
