package main

import (
	"os"
	"reflect"
	"runtime"
	"strings"
)

//go:noinline
func caller(available bool) {
//line pcln-callsite.go:123
	pc, file, line, ok := runtime.Caller(0)
	if !ok || pc == 0 {
		panic("lost caller PC")
	}
	if available && (!strings.HasSuffix(file, "pcln-callsite.go") || line != 123) {
		println(file, line)
		panic("lost caller source")
	}
	if !available && strings.HasSuffix(file, "pcln-callsite.go") {
		panic("unexpected external source")
	}
	if available {
		fn := runtime.FuncForPC(pc)
		if fn == nil || !strings.HasSuffix(fn.Name(), ".caller") {
			panic("lost caller name")
		}
		fn = runtime.FuncForPC(reflect.ValueOf(caller).Pointer())
		if fn == nil || !strings.HasSuffix(fn.Name(), ".caller") {
			panic("lost function value")
		}
	}
}

func main() {
	mode := "available"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode == "panic" {
		panic("pcln-fatal")
	}
	if mode == "no-symbols" {
		println("pcln-ok")
		return
	}
	available := mode != "missing"
	if mode == "cached-success" || mode == "cached-failure" {
		path := os.Args[2]
		if mode == "cached-success" {
			caller(true)
		}
		if err := os.Rename(path, path+".saved"); err != nil {
			panic(err)
		}
		caller(mode == "cached-success")
		if err := os.Rename(path+".saved", path); err != nil {
			panic(err)
		}
		caller(mode == "cached-success")
	} else if mode == "concurrent" {
		done := make(chan struct{}, 33)
		go func() {
			for range 3 {
				runtime.GC()
			}
			done <- struct{}{}
		}()
		for range 32 {
			go func() { caller(true); done <- struct{}{} }()
		}
		for range 33 {
			<-done
		}
	} else {
		caller(available)
	}
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	if n == 0 {
		panic("lost callers")
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		_, more := frames.Next()
		if !more {
			break
		}
	}
	value := 0
	done := make(chan struct{})
	go func() { value = 42; close(done) }()
	<-done
	if value != 42 {
		panic("application did not continue")
	}
	println("pcln-ok")
}
