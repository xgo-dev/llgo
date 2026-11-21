package main

import (
	"fmt"
	"os"
	"reflect"
	"time"
)

type entry struct {
	Value int `json:"value"`
}

func (e entry) Read() int { return e.Value }

func main() {
	// The console path must still observe changes to the public file.
	stdout := os.Stdout
	os.Stdout = os.Stderr
	n, err := fmt.Printf("redirected\n")
	if n != len("redirected\n") || err != nil {
		panic("redirected console write failed")
	}
	os.Stdout = nil
	n, err = fmt.Printf("inaccessible console\n")
	if n != 0 || err != os.ErrInvalid {
		panic("invalid console write succeeded")
	}
	os.Stdout = stdout
	if err := stdout.SetWriteDeadline(time.Now()); err != os.ErrNoDeadline {
		panic("console reported a write deadline")
	}

	// Reflected field and method queries keep their metadata on a closed link.
	value := entry{Value: 42}
	object := reflect.ValueOf(&value).Elem()
	field, found := object.Type().FieldByName("Value")
	method, hasMethod := object.Type().MethodByName("Read")
	if !found || field.Tag.Get("json") != "value" || !hasMethod || method.Type.NumOut() != 1 {
		panic("missing reflected metadata")
	}
	object.FieldByName("Value").SetInt(17)
	var boxed any = value
	if boxed.(interface{ Read() int }).Read() != 17 {
		panic("bad reflected field update")
	}
	const key = "LLGO_EMBEDDED_TEST_ENV"
	if err := os.Setenv(key, "hello"); err != nil || os.Getenv(key) != "hello" {
		panic("could not set embedded environment")
	}
	if err := os.Unsetenv(key); err != nil {
		panic("could not unset embedded environment")
	}
	if _, found := os.LookupEnv(key); found {
		panic("embedded environment retained deleted key")
	}
	println("embedded output and reflection ok")
}
