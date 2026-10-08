//go:build js && wasm && llgo.wasm.workers

package main

import (
	"reflect"
	"runtime"
)

func main() {
	const count = 20
	done := make(chan int, count)
	stop, gcDone := make(chan bool), make(chan bool)
	go func() {
		defer close(gcDone)
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
				runtime.Gosched()
			}
		}
	}()
	for i := 0; i < count; i++ {
		fn := reflect.MakeFunc(reflect.TypeOf((func(*int))(nil)), func(args []reflect.Value) []reflect.Value {
			if len(args) != 1 || !args[0].IsNil() {
				panic("invalid MakeFunc arguments")
			}
			runtime.Gosched()
			done <- 1
			return nil
		}).Interface().(func(*int))
		go fn(nil)
	}
	for i := 0; i < count; i++ {
		if <-done != 1 {
			panic("invalid MakeFunc result")
		}
	}
	close(stop)
	<-gcDone
	println("wasm reflect workers ok")
}
