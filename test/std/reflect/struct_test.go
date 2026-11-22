package reflect_test

import (
	"reflect"
	"testing"
)

// Test struct field operations
func TestStructFieldOperations(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p := Person{Name: "Alice", Age: 30}
	v := reflect.ValueOf(p)

	// Test NumField
	if v.NumField() != 2 {
		t.Errorf("NumField should be 2, got %d", v.NumField())
	}

	// Test Field
	nameField := v.Field(0)
	if nameField.String() != "Alice" {
		t.Errorf("Field(0) should be 'Alice', got %q", nameField.String())
	}

	// Test FieldByName
	ageField := v.FieldByName("Age")
	if !ageField.IsValid() {
		t.Fatal("FieldByName('Age') should be valid")
	}
	if ageField.Int() != 30 {
		t.Errorf("FieldByName('Age') should be 30, got %d", ageField.Int())
	}
}

// Test Type methods
func TestTypeMethods(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	typ := reflect.TypeOf(Person{})

	// Test NumField
	if typ.NumField() != 2 {
		t.Errorf("Type.NumField should be 2, got %d", typ.NumField())
	}

	// Test Field
	field := typ.Field(0)
	if field.Name != "Name" {
		t.Errorf("Field(0).Name should be 'Name', got %q", field.Name)
	}

	// Test FieldByName
	field, ok := typ.FieldByName("Age")
	if !ok {
		t.Fatal("FieldByName('Age') should be found")
	}
	if field.Name != "Age" {
		t.Errorf("Field.Name should be 'Age', got %q", field.Name)
	}
}

// Test StructField and StructTag
func TestStructFieldAndTag(t *testing.T) {
	type Tagged struct {
		Field1 string `json:"field1" xml:"f1"`
		Field2 int    `json:"field2,omitempty"`
	}

	typ := reflect.TypeOf(Tagged{})
	field := typ.Field(0)

	// Test StructField
	if field.Name != "Field1" {
		t.Errorf("StructField.Name should be 'Field1', got %q", field.Name)
	}

	// Test StructTag.Get
	tag := field.Tag.Get("json")
	if tag != "field1" {
		t.Errorf("Tag.Get('json') should be 'field1', got %q", tag)
	}

	// Test StructTag.Lookup
	val, ok := field.Tag.Lookup("xml")
	if !ok {
		t.Fatal("Tag.Lookup('xml') should find tag")
	}
	if val != "f1" {
		t.Errorf("Tag.Lookup('xml') should be 'f1', got %q", val)
	}
}

// Test StructField.IsExported
func TestStructFieldIsExported(t *testing.T) {
	type T struct {
		Exported   int
		unexported int
	}

	typ := reflect.TypeOf(T{})

	field0 := typ.Field(0)
	if !field0.IsExported() {
		t.Error("Exported field should be exported")
	}

	field1 := typ.Field(1)
	if field1.IsExported() {
		t.Error("unexported field should not be exported")
	}
}

// Test Value.FieldByIndex
func TestValueFieldByIndex(t *testing.T) {
	type Inner struct {
		Value int
	}
	type Outer struct {
		Inner
	}

	o := Outer{Inner{42}}
	v := reflect.ValueOf(o)

	field := v.FieldByIndex([]int{0, 0})
	if field.Int() != 42 {
		t.Errorf("FieldByIndex should get 42, got %d", field.Int())
	}
}

// Test FieldByNameFunc
func TestValueFieldByNameFunc(t *testing.T) {
	type Person struct {
		FirstName string
		LastName  string
	}

	p := Person{FirstName: "John", LastName: "Doe"}
	v := reflect.ValueOf(p)

	field := v.FieldByNameFunc(func(name string) bool {
		return name == "LastName"
	})

	if !field.IsValid() {
		t.Fatal("FieldByNameFunc should find LastName")
	}
	if field.String() != "Doe" {
		t.Errorf("FieldByNameFunc should get 'Doe', got %q", field.String())
	}
}

func TestPromotedFieldByName(t *testing.T) {
	type Inner struct {
		Prefix byte
		Number int `json:"number"`
	}
	type Outer struct {
		Prefix string
		*Inner
	}
	outer := Outer{Inner: &Inner{Number: 42}}
	v := reflect.ValueOf(&outer).Elem()
	match := func(name string) bool { return name == "Number" }
	for _, field := range []reflect.Value{v.FieldByName("Number"), v.FieldByNameFunc(match)} {
		if !field.CanSet() || field.Int() != 42 {
			t.Fatalf("promoted value = %v, want settable 42", field)
		}
	}
	for _, lookup := range []func() (reflect.StructField, bool){
		func() (reflect.StructField, bool) { return v.Type().FieldByName("Number") },
		func() (reflect.StructField, bool) { return v.Type().FieldByNameFunc(match) },
	} {
		field, ok := lookup()
		want := reflect.TypeOf(Inner{}).Field(1)
		if !ok || field.Type != want.Type || field.Offset != want.Offset || field.Tag != want.Tag ||
			len(field.Index) != 2 || field.Index[0] != 1 || field.Index[1] != 1 {
			t.Fatalf("promoted field = %+v, found %v", field, ok)
		}
		field.Index[0] = 0
		again, _ := lookup()
		if again.Index[0] != 1 {
			t.Fatal("lookups share writable index storage")
		}
	}
	v.FieldByName("Number").SetInt(43)
	if outer.Number != 43 {
		t.Fatal("promoted field does not address original value")
	}

	type Left struct{ *Inner }
	type Right struct{ *Inner }
	v = reflect.ValueOf(struct {
		Left
		Right
	}{})
	if v.FieldByName("Number").IsValid() || v.FieldByNameFunc(match).IsValid() {
		t.Fatal("ambiguous promoted value was returned")
	}
	if _, ok := v.Type().FieldByName("Number"); ok {
		t.Fatal("ambiguous promoted type field was returned")
	}

	defer func() {
		if recover() == nil {
			t.Error("lookup through a nil embedded pointer did not panic")
		}
	}()
	reflect.ValueOf(Outer{}).FieldByName("Number")
}

func TestPromotedFieldByNameTraversal(t *testing.T) {
	type A struct{}
	type B struct{}
	type C struct{}
	type D struct{}
	type E struct{}
	type F struct{}
	type G struct{}
	type H struct{}
	type Cycle struct{ *Cycle }
	type Leaf struct{ Number int }
	type Left struct{ Leaf }
	type Right struct{ Leaf }
	type Branch struct{ Right }
	type Wide struct {
		A
		B
		C
		D
		E
		F
		G
		H
		*Cycle
		Left
	}

	lookup := func(typ reflect.Type) (reflect.StructField, bool) {
		return typ.FieldByNameFunc(func(name string) bool { return name == "Number" })
	}
	field, found := lookup(reflect.TypeOf(Wide{}))
	if !found || len(field.Index) != 3 || field.Index[0] != 9 ||
		field.Index[1] != 0 || field.Index[2] != 0 {
		t.Fatalf("promoted field index = %v, found %v", field.Index, found)
	}
	if _, found := lookup(reflect.TypeOf(struct {
		Wide
		Branch
	}{})); found {
		t.Fatal("ambiguous field returned after queue growth")
	}
}
