package main

import (
	"fmt"
	"os"
	"reflect"
)

type reading struct {
	Count int `json:"count"`
}

func (r reading) Read() int { return r.Count }

//go:noinline
func writeValue(format string, value any) (int, error) {
	return fmt.Printf(format, value)
}

func main() {
	value := reading{Count: 42}
	object := reflect.ValueOf(&value).Elem()
	field, ok := object.Type().FieldByName("Count")
	if !ok || field.Tag.Get("json") != "count" {
		panic("lost field information")
	}
	object.FieldByIndex(field.Index).SetInt(17)
	method := object.MethodByName("Read")
	if !method.IsValid() || method.Call(nil)[0].Int() != 17 {
		panic("lost reflected method")
	}
	var boxed any = value
	if boxed.(interface{ Read() int }).Read() != 17 {
		panic("lost interface method")
	}
	stdout := os.Stdout
	os.Stdout = os.Stderr
	n, err := writeValue("redirected %d\n", value.Count)
	if n != len("redirected 17\n") || err != nil {
		panic("redirected write failed")
	}
	os.Stdout = nil
	n, err = writeValue("%d\n", value.Count)
	if n != 0 || err != os.ErrInvalid {
		panic("invalid write succeeded")
	}
	os.Stdout = stdout
	println("native output and reflection ok")
}
