// LITTEST
package main

type T struct {
	s int
}

func f() *T {
	return nil
}

// CHECK: ; Function Attrs: null_pointer_is_valid
// CHECK-LABEL: define void @"main.init#1"() #0 {
func init() {
	println("init")
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			println("recover", e.Error())
		}
	}()
	// CHECK: %[[NIL_T:[0-9]+]] = call ptr @main.f()
	// CHECK: %[[FIELD:[0-9]+]] = getelementptr inbounds %main.T, ptr %[[NIL_T]], i32 0, i32 0
	// CHECK: load i64, ptr %[[FIELD]]
	println(f().s)
}

func main() {
	println("main")
}

// CHECK: attributes #0 = { null_pointer_is_valid "frame-pointer"="non-leaf" }
