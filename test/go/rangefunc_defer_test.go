package gotest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func rangefuncDeferYield4(yield func(int) bool) {
	_ = yield(1) && yield(2) && yield(3) && yield(4)
}

func TestRangeFuncDefersRunAfterOuterFunction(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(-1)
		for i := range rangefuncDeferYield4 {
			defer save(i)
		}
		defer save(5)
	}()

	want := []int{5, 4, 3, 2, 1, -1}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order = %v, want %v", saved, want)
	}
}

func TestRangeFuncDefersRunAfterEarlyBreak(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		for i := range rangefuncDeferYield4 {
			defer save(i)
			if i == 2 {
				break
			}
		}
		defer save(9)
	}()

	want := []int{9, 2, 1}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after break = %v, want %v", saved, want)
	}
}

func TestRangeFuncDefersRunAfterContinue(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(9)
		for i := range rangefuncDeferYield4 {
			if i%2 == 0 {
				continue
			}
			defer save(i)
		}
	}()

	want := []int{3, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after continue = %v, want %v", saved, want)
	}
}

func TestRangeFuncDefersRunAfterReturn(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(9)
		for i := range rangefuncDeferYield4 {
			defer save(i)
			if i == 2 {
				return
			}
		}
		defer save(8)
	}()

	want := []int{2, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after return = %v, want %v", saved, want)
	}
}

func TestRangeFuncDefersRunAfterLabeledBreak(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(9)
	outer:
		for range [2]int{} {
			for i := range rangefuncDeferYield4 {
				defer save(i)
				if i == 2 {
					break outer
				}
			}
		}
	}()

	want := []int{2, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after labeled break = %v, want %v", saved, want)
	}
}

func TestRangeFuncDefersRunAfterLabeledContinue(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(9)
	outer:
		for round := range [2]int{} {
			for i := range rangefuncDeferYield4 {
				if i == 2 {
					continue outer
				}
				defer save(round*10 + i)
			}
		}
	}()

	want := []int{11, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after labeled continue = %v, want %v", saved, want)
	}
}

func TestRangeFuncDefersRunAfterGoto(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(9)
		for i := range rangefuncDeferYield4 {
			defer save(i)
			if i == 2 {
				goto done
			}
		}
		defer save(8)
	done:
	}()

	want := []int{2, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after goto = %v, want %v", saved, want)
	}
}

func rangefuncDeferNestedOuter(yield func(int) bool) {
	_ = yield(1) && yield(2)
}

func rangefuncDeferNestedInner(base int) func(func(int) bool) {
	return func(yield func(int) bool) {
		_ = yield(base*10+1) && yield(base*10+2)
	}
}

func TestNestedRangeFuncDefers(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(99)
		for i := range rangefuncDeferNestedOuter {
			defer save(i)
			for j := range rangefuncDeferNestedInner(i) {
				if j%10 == 2 {
					break
				}
				defer save(j)
			}
		}
	}()

	want := []int{21, 2, 11, 1, 99}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("nested defer order = %v, want %v", saved, want)
	}
}

func TestNestedRangeFuncDefersRunAfterReturn(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	func() {
		defer save(99)
		for i := range rangefuncDeferNestedOuter {
			defer save(i)
			for j := range rangefuncDeferNestedInner(i) {
				defer save(j)
				if j == 12 {
					return
				}
			}
		}
	}()

	want := []int{12, 11, 1, 99}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("nested defer order after return = %v, want %v", saved, want)
	}
}

func TestNestedRangeFuncDefersRunAfterPanicRecover(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	var recovered any
	func() {
		defer func() {
			recovered = recover()
			save(99)
		}()
		for i := range rangefuncDeferNestedOuter {
			defer save(i)
			for j := range rangefuncDeferNestedInner(i) {
				defer save(j)
				if j == 12 {
					panic("nested boom")
				}
			}
		}
	}()

	want := []int{12, 11, 1, 99}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("nested defer order after panic = %v, want %v", saved, want)
	}
	if recovered != "nested boom" {
		t.Fatalf("recover = %v, want nested boom", recovered)
	}
}

func TestRangeFuncDefersRunAfterPanicRecover(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	var recovered any
	func() {
		defer func() {
			recovered = recover()
			save(9)
		}()
		for i := range rangefuncDeferYield4 {
			defer save(i)
			if i == 2 {
				panic("boom")
			}
		}
	}()

	want := []int{2, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after panic = %v, want %v", saved, want)
	}
	if recovered != "boom" {
		t.Fatalf("recover = %v, want boom", recovered)
	}
}

func TestRangeFuncDeferredRecoverInLoopBody(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	var recovered any
	func() {
		defer save(9)
		for i := range rangefuncDeferYield4 {
			defer func(v int) {
				if r := recover(); r != nil {
					recovered = r
					save(100 + v)
					return
				}
				save(v)
			}(i)
			if i == 2 {
				panic("boom")
			}
		}
	}()

	want := []int{102, 1, 9}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("defer order after in-body recover = %v, want %v", saved, want)
	}
	if recovered != "boom" {
		t.Fatalf("recover = %v, want boom", recovered)
	}
}

func TestNestedRangeFuncDeferredRecoverInLoopBody(t *testing.T) {
	var saved []int
	save := func(v int) {
		saved = append(saved, v)
	}

	var recovered any
	func() {
		defer save(99)
		for i := range rangefuncDeferNestedOuter {
			defer save(i)
			for j := range rangefuncDeferNestedInner(i) {
				defer func(v int) {
					if r := recover(); r != nil {
						recovered = r
						save(100 + v)
						return
					}
					save(v)
				}(j)
				if j == 12 {
					panic("nested boom")
				}
			}
		}
	}()

	want := []int{112, 11, 1, 99}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("nested defer order after in-body recover = %v, want %v", saved, want)
	}
	if recovered != "nested boom" {
		t.Fatalf("recover = %v, want nested boom", recovered)
	}
}

func TestRangeFuncRejectsYieldAfterLoopExit(t *testing.T) {
	var recovered any

	func() {
		defer func() {
			recovered = recover()
		}()

		for i := range func(yield func(int) bool) {
			yield(1)
			yield(2)
		} {
			if i != 1 {
				t.Fatalf("got %d, want 1", i)
			}
			break
		}
	}()

	msg := fmt.Sprint(recovered)
	if !(strings.Contains(msg, "continued iteration") && strings.Contains(msg, "returned false")) &&
		msg != "yield function called after range loop exit" {
		t.Fatalf("panic = %q, want range loop exit panic", msg)
	}
}

func TestRangeFuncRejectsYieldAfterBodyPanic(t *testing.T) {
	var recovered any

	func() {
		defer func() {
			recovered = recover()
		}()

		for range func(yield func(int) bool) {
			defer func() {
				if recover() == "loop panic" {
					yield(2)
				}
			}()
			yield(1)
		} {
			panic("loop panic")
		}
	}()

	msg := fmt.Sprint(recovered)
	if !(strings.Contains(msg, "continued iteration") && strings.Contains(msg, "loop body panic")) &&
		msg != "yield function called after range loop exit" {
		t.Fatalf("panic = %q, want range body panic", msg)
	}
}
