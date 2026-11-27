// LITTEST
// Scope: common
package main

type pair struct {
	x int
	y int
}

var state pair
var calls int

// The right-hand side must be completely evaluated before state is replaced.
// CHECK-LABEL: define void @main.assignGlobal(){{.*}} {
// CHECK: call i64 @main.first()
// CHECK: call i64 @main.second()
// CHECK: store %main.pair %{{[0-9]+}}, ptr @main.state
func assignGlobal() {
	state = pair{first(), second()}
	if state != (pair{1, 2}) {
		panic("global composite assignment produced the wrong value")
	}
}

func first() int {
	state.x = 42
	return 1
}

func second() int {
	if state.x != 42 {
		panic("composite assignment committed a field before evaluating its right-hand side")
	}
	return 2
}

// A nil destination must not panic until both composite-literal operands have
// been evaluated.
// CHECK-LABEL: define void @main.assignNil(){{.*}} {
// CHECK: call i64 @main.count()
// CHECK: call i64 @main.count()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"
func assignNil() {
	defer func() {
		if recover() == nil {
			panic("nil composite assignment did not panic")
		}
		if calls != 2 {
			panic("nil composite assignment panicked before evaluating its right-hand side")
		}
	}()

	var dst *pair
	*dst = pair{count(), count()}
}

func count() int {
	calls++
	return calls
}

func main() {
	assignGlobal()
	assignNil()
	println("ok")
}
