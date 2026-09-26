package gotest

import (
	"reflect"
	"runtime"
	"testing"
)

func TestReflectMakeFuncGoroutineStartup(t *testing.T) {
	for _, tc := range []struct {
		name    string
		withArg bool
	}{
		{"pointer argument", true},
		{"zero arguments", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testReflectMakeFuncGoroutineStartup(t, tc.withArg)
		})
	}
}

func testReflectMakeFuncGoroutineStartup(t *testing.T, withArg bool) {
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
				// Keep this a GC/startup race without requiring asynchronous
				// preemption from a single-threaded cooperative runtime.
				runtime.Gosched()
			}
		}
	}()
	defer func() {
		close(stopGC)
		<-gcDone
	}()

	const n = 20
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		if withArg {
			f := reflect.MakeFunc(reflect.TypeOf((func(*int))(nil)), func(args []reflect.Value) []reflect.Value {
				if len(args) != 1 || !args[0].IsNil() {
					panic("bad reflect MakeFunc pointer argument")
				}
				done <- struct{}{}
				return nil
			}).Interface().(func(*int))
			go f(nil)
		} else {
			f := reflect.MakeFunc(reflect.TypeOf((func())(nil)), func(args []reflect.Value) []reflect.Value {
				if len(args) != 0 {
					panic("bad reflect MakeFunc zero-argument call")
				}
				done <- struct{}{}
				return nil
			}).Interface().(func())
			go f()
		}
	}

	for i := 0; i < n; i++ {
		<-done
	}
}

func TestReflectMakeFuncGoroutineGC(t *testing.T) {
	done := make(chan struct{})
	f := reflect.MakeFunc(reflect.TypeOf((func())(nil)), func(args []reflect.Value) []reflect.Value {
		if len(args) != 0 {
			panic("bad reflect MakeFunc zero-argument call")
		}
		runtime.GC()
		close(done)
		return nil
	}).Interface().(func())
	go f()
	<-done
}
