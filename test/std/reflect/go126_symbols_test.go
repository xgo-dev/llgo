//go:build go1.26

package reflect_test

import (
	"reflect"
	"testing"
)

type go126Value struct {
	Name  string
	Count int
}

func (v go126Value) Summary(prefix string) string {
	return prefix + v.Name
}

func TestValueFieldsAndMethods(t *testing.T) {
	value := reflect.ValueOf(go126Value{Name: "llgo", Count: 2})
	fields := make(map[string]any)
	for field, fieldValue := range value.Fields() {
		fields[field.Name] = fieldValue.Interface()
	}
	if fields["Name"] != "llgo" || fields["Count"] != 2 || len(fields) != 2 {
		t.Fatalf("Fields returned %#v", fields)
	}

	called := false
	for method, methodValue := range value.Methods() {
		if method.Name == "Summary" {
			called = true
			result := methodValue.Call([]reflect.Value{reflect.ValueOf("value:")})
			if len(result) != 1 || result[0].String() != "value:llgo" {
				t.Fatalf("bound method result = %#v", result)
			}
		}
	}
	if !called {
		t.Fatal("Methods did not return Summary")
	}
}
