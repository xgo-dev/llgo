// LITTEST
package main

import "github.com/goplus/llgo/cl/_testgo/tlsgls/state"

func printValues(owner string) {
	// CHECK-LABEL: define void @"{{.*}}tlsgls.printValues"
	// CHECK: call { i64, i64, i64, i64 } @"{{.*}}tlsgls/state.Values"()
	tls, gls, pointer, sequence := state.Values()
	// CHECK: call { i64, i64 } @"{{.*}}tlsgls/state.PoisonValues"()
	poison, attempts := state.PoisonValues()
	// CHECK: call { i64, i64 } @"{{.*}}tlsgls/state.LateValues"()
	late, lateAttempts := state.LateValues()
	// CHECK: call i64 @"{{.*}}tlsgls/state.RecursiveValue"()
	println(owner, tls, gls, pointer, sequence, poison, attempts, state.RecursiveValue(), late, lateAttempts)
}

func run(owner string, done chan bool) {
	printValues(owner)
	done <- true
}

func main() {
	printValues("main")
	done := make(chan bool)
	go run("g1", done)
	<-done
	go run("g2", done)
	<-done
	printValues("main")
}
