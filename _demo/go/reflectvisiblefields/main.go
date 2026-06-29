package main

import (
	"fmt"
	"reflect"
)

// Define a struct with embedded anonymous fields and various visibility
type Inner struct {
	InnerExported   int    // exported field in embedded struct
	innerUnexported string // unexported field in embedded struct
}

type Outer struct {
	OuterField        string
	Inner             // embedded (anonymous) struct
	AnotherExported   float64
	anotherUnexported bool
}

func main() {
	t := reflect.TypeOf(Outer{})

	// Get all visible fields
	fields := reflect.VisibleFields(t)

	fmt.Printf("Struct: %s\n\n", t.Name())
	fmt.Printf("Total visible fields: %d\n\n", len(fields))

	for i, f := range fields {
		fmt.Printf("[%d] Name: %-18s Index: %-4v Anonymous: %-5v Type: %-10v PkgPath: %q\n",
			i,
			f.Name,
			f.Index,
			f.Anonymous,
			f.Type,
			f.PkgPath,
		)
	}

	fmt.Println("\n--- Access field values via FieldByIndex ---")

	o := Outer{
		OuterField:        "hello",
		Inner:             Inner{InnerExported: 42, innerUnexported: "hidden"},
		AnotherExported:   3.14,
		anotherUnexported: true,
	}

	v := reflect.ValueOf(o)

	// Access each field using its Index
	for _, f := range fields {
		fieldVal := v.FieldByIndex(f.Index)
		if fieldVal.CanInterface() {
			fmt.Printf("%-18s = %v\n", f.Name, fieldVal.Interface())
		} else {
			fmt.Printf("%-18s = <unexported: %v>\n", f.Name, fieldVal)
		}
	}
}
