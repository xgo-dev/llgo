package reflect_test

import (
	"math/big"
	"reflect"
	"testing"
)

// Test Type.Name and Type.String
func TestTypeName(t *testing.T) {
	type MyInt int
	typ := reflect.TypeOf(MyInt(0))

	name := typ.Name()
	if name != "MyInt" {
		t.Errorf("Type.Name should be 'MyInt', got %q", name)
	}

	str := typ.String()
	if str == "" {
		t.Error("Type.String should not be empty")
	}
}

// Test Type.Kind
func TestTypeKind(t *testing.T) {
	tests := []struct {
		value any
		kind  reflect.Kind
	}{
		{42, reflect.Int},
		{"hello", reflect.String},
		{true, reflect.Bool},
		{3.14, reflect.Float64},
		{[]int{}, reflect.Slice},
		{[3]int{}, reflect.Array},
		{map[string]int{}, reflect.Map},
		{struct{}{}, reflect.Struct},
	}

	for _, tt := range tests {
		typ := reflect.TypeOf(tt.value)
		if typ.Kind() != tt.kind {
			t.Errorf("TypeOf(%v).Kind() = %v, want %v", tt.value, typ.Kind(), tt.kind)
		}
	}
}

func TestTypeForFuncMatchesTypeOf(t *testing.T) {
	type Action func()
	type Predicate func() bool

	tests := []struct {
		got  reflect.Type
		want reflect.Type
	}{
		{reflect.TypeFor[func()](), reflect.TypeOf((func())(nil))},
		{reflect.TypeFor[func() bool](), reflect.TypeOf((func() bool)(nil))},
		{reflect.TypeFor[Action](), reflect.TypeOf((Action)(nil))},
		{reflect.TypeFor[Predicate](), reflect.TypeOf((Predicate)(nil))},
	}
	for _, tt := range tests {
		if tt.got.Kind() != reflect.Func {
			t.Errorf("TypeFor kind = %v; want Func for %s", tt.got.Kind(), tt.got)
		}
		if tt.got != tt.want {
			t.Errorf("TypeFor = %s kind %v; TypeOf = %s kind %v", tt.got, tt.got.Kind(), tt.want, tt.want.Kind())
		}
	}

	called := 0
	fn := reflect.MakeFunc(reflect.TypeFor[func()](), func([]reflect.Value) []reflect.Value {
		called++
		return nil
	}).Interface().(func())
	fn()
	if called != 1 {
		t.Fatalf("MakeFunc(TypeFor[func()]) called %d times; want 1", called)
	}
}

func TestFuncOfInternsSignaturesAcrossGoroutines(t *testing.T) {
	in := []reflect.Type{reflect.TypeFor[[]int]()}
	out := []reflect.Type{reflect.TypeFor[int]()}
	want := reflect.TypeFor[func(...int) int]()

	const workers = 4
	types := make(chan reflect.Type, workers)
	for range workers {
		go func() {
			typ := reflect.FuncOf(in, out, true)
			fn := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(args[0].Len())}
			})
			got := fn.CallSlice([]reflect.Value{reflect.ValueOf([]int{4, 5})})[0].Int()
			if got != 2 || fn.Type() != typ || typ != want {
				t.Errorf("MakeFunc type = %v, FuncOf type = %v, result = %d", fn.Type(), typ, got)
			}
			types <- reflect.FuncOf(in, out, false)
		}()
	}
	plain := <-types
	if plain == want || plain.IsVariadic() {
		t.Fatalf("non-variadic signature interned as %v", plain)
	}
	for range workers - 1 {
		if got := <-types; got != plain {
			t.Errorf("FuncOf reused distinct descriptors for %v and %v", got, plain)
		}
	}
}

func TestTypeForMatchesStructFieldType(t *testing.T) {
	type holder struct {
		Value *big.Int
	}

	direct := reflect.TypeFor[*big.Int]()
	repeated := reflect.TypeFor[*big.Int]()
	field := reflect.TypeFor[holder]().Field(0).Type
	if direct != repeated {
		t.Fatalf("repeated TypeFor returned distinct types: %v and %v", direct, repeated)
	}
	if direct != field {
		t.Fatalf("TypeFor type %v differs from struct field type %v", direct, field)
	}
	value := reflect.ValueOf(&holder{}).Elem().Field(0).Addr().Interface()
	if _, ok := value.(**big.Int); !ok {
		t.Fatalf("reflected field address has type %T, want **big.Int", value)
	}
}
