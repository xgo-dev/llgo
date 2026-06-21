package gotest

import (
	"os"
	"path/filepath"
	"testing"
)

const reflectTypeMetadataMakeFuncProbe = `package main

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

type foo struct {
	bar int
}

var blankFieldEvalCount int

func blankFieldValue() int {
	blankFieldEvalCount++
	return 2
}

func boolLoop(next func() bool) bool {
	for b := next(); b; b = next() {
		return true
	}
	return false
}

func errString(v any) string {
	if s, ok := v.(interface{ Error() string }); ok {
		return s.Error()
	}
	return reflect.ValueOf(v).String()
}

func expectReflectMakeFuncPanic() {
	defer func() {
		err := recover()
		if err == nil {
			panic("MakeFunc call did not panic")
		}
		if got := errString(err); !strings.HasPrefix(got, "reflect:") {
			panic("MakeFunc panic missing reflect prefix: " + got)
		}
	}()

	fn := reflect.MakeFunc(reflect.TypeOf(func() error { return nil }), func([]reflect.Value) []reflect.Value {
		var out [1]reflect.Value
		return out[:]
	}).Interface().(func() error)
	_ = fn()
}

func main() {
	if got := reflect.ValueOf(foo{}).Type().Field(0).PkgPath; got != "main" {
		panic(fmt.Sprintf("field PkgPath = %q, want main", got))
	}
	if got := reflect.TypeOf(unsafe.Pointer(nil)).PkgPath(); got != "unsafe" {
		panic(fmt.Sprintf("unsafe.Pointer PkgPath = %q, want unsafe", got))
	}

	x := struct{ a, _, c int }{1, blankFieldValue(), 3}
	if blankFieldEvalCount != 1 {
		panic(fmt.Sprintf("blank field initializer evaluated %d times, want 1", blankFieldEvalCount))
	}
	if got := reflect.ValueOf(x).Field(1).Int(); got != 0 {
		panic(fmt.Sprintf("blank field reflect value = %d, want 0", got))
	}

	expectReflectMakeFuncPanic()

	nextFalse := reflect.MakeFunc(reflect.TypeOf((func() bool)(nil)), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(false)}
	})
	if got := reflect.ValueOf(boolLoop).Call([]reflect.Value{nextFalse})[0].Bool(); got {
		panic(fmt.Sprintf("false MakeFunc loop result = %v, want false", got))
	}

	nextTrue := reflect.MakeFunc(reflect.TypeOf((func() bool)(nil)), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(true)}
	})
	if got := reflect.ValueOf(boolLoop).Call([]reflect.Value{nextTrue})[0].Bool(); !got {
		panic(fmt.Sprintf("true MakeFunc loop result = %v, want true", got))
	}
}
`

func TestReflectTypeMetadataMakeFuncProbe(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(reflectTypeMetadataMakeFuncProbe), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, dir, "run", mainFile)

	root := findLLGoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	runGoCmd(t, root, "run", "./cmd/llgo", "run", mainFile)
}
