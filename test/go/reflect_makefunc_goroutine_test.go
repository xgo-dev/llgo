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

// Exercise the interface-result allocation path from GOROOT's issue39541.
func TestReflectMakeFuncConcurrentInterfaceResults(t *testing.T) {
	type result struct{ x, y int }
	f := reflect.MakeFunc(reflect.TypeOf((func() any)(nil)), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(result{1, 2})}
	}).Interface().(func() any)
	const workers = 8
	done := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		go func() {
			valid := true
			for j := 0; j < 512; j++ {
				if f() != (result{1, 2}) {
					valid = false
				}
			}
			done <- valid
		}()
	}
	for i := 0; i < workers; i++ {
		if !<-done {
			t.Fatal("concurrent MakeFunc call returned an incorrect interface value")
		}
	}
}
