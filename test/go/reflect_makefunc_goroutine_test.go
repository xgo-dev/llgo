package gotest

import (
	"reflect"
	"runtime"
	"testing"
)

func TestReflectMakeFuncGoroutineStartup(t *testing.T) {
	oldProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldProcs)

	stopGC := make(chan struct{})
	gcDone := make(chan struct{})
	go func() {
		defer close(gcDone)
		for {
			select {
			case <-stopGC:
				return
			default:
				runtime.GC()
			}
		}
	}()
	defer func() {
		close(stopGC)
		<-gcDone
	}()

	const n = 100
	done := make(chan struct{}, n*2)
	for i := 0; i < n; i++ {
		f := reflect.MakeFunc(reflect.TypeOf((func(*int))(nil)), func(args []reflect.Value) []reflect.Value {
			if len(args) != 1 || !args[0].IsNil() {
				panic("bad reflect MakeFunc pointer argument")
			}
			done <- struct{}{}
			return nil
		}).Interface().(func(*int))
		go f(nil)

		g := reflect.MakeFunc(reflect.TypeOf((func())(nil)), func(args []reflect.Value) []reflect.Value {
			if len(args) != 0 {
				panic("bad reflect MakeFunc zero-argument call")
			}
			done <- struct{}{}
			return nil
		}).Interface().(func())
		go g()
	}

	for i := 0; i < n*2; i++ {
		<-done
	}
}
