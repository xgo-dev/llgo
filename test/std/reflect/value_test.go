package reflect_test

import (
	"reflect"
	"testing"
	"unsafe"
)

// Test Value.Type
func TestValueType(t *testing.T) {
	v := reflect.ValueOf(42)
	typ := v.Type()
	if typ == nil {
		t.Fatal("Value.Type returned nil")
	}
	if typ.Kind() != reflect.Int {
		t.Errorf("Type.Kind should be Int, got %v", typ.Kind())
	}
}

// Test basic value operations
func TestValueOperations(t *testing.T) {
	// Test with int
	v := reflect.ValueOf(42)
	if v.Int() != 42 {
		t.Errorf("Int should be 42, got %d", v.Int())
	}

	// Test with string
	v = reflect.ValueOf("hello")
	if v.String() != "hello" {
		t.Errorf("String should be 'hello', got %q", v.String())
	}

	// Test with bool
	v = reflect.ValueOf(true)
	if !v.Bool() {
		t.Error("Bool should be true")
	}

	// Test with float64
	v = reflect.ValueOf(3.14)
	if v.Float() != 3.14 {
		t.Errorf("Float should be 3.14, got %f", v.Float())
	}
}

// Test Value.Set operations
func TestValueSet(t *testing.T) {
	// Test SetInt
	var i int
	v := reflect.ValueOf(&i).Elem()
	v.SetInt(42)
	if i != 42 {
		t.Errorf("SetInt: expected 42, got %d", i)
	}

	// Test SetBool
	var b bool
	v = reflect.ValueOf(&b).Elem()
	v.SetBool(true)
	if !b {
		t.Error("SetBool: expected true")
	}

	// Test SetFloat
	var f float64
	v = reflect.ValueOf(&f).Elem()
	v.SetFloat(3.14)
	if f != 3.14 {
		t.Errorf("SetFloat: expected 3.14, got %f", f)
	}

	// Test SetString
	var s string
	v = reflect.ValueOf(&s).Elem()
	v.SetString("hello")
	if s != "hello" {
		t.Errorf("SetString: expected 'hello', got %q", s)
	}

	// Test SetUint
	var u uint
	v = reflect.ValueOf(&u).Elem()
	v.SetUint(100)
	if u != 100 {
		t.Errorf("SetUint: expected 100, got %d", u)
	}

	// Test SetComplex
	var c complex128
	v = reflect.ValueOf(&c).Elem()
	v.SetComplex(1 + 2i)
	if c != 1+2i {
		t.Errorf("SetComplex: expected (1+2i), got %v", c)
	}
}

// Test Value.CanSet and CanAddr
func TestValueCanSetCanAddr(t *testing.T) {
	// Direct value cannot be set
	v := reflect.ValueOf(42)
	if v.CanSet() {
		t.Error("Direct value should not be settable")
	}

	// Elem of pointer can be set
	i := 42
	v = reflect.ValueOf(&i).Elem()
	if !v.CanSet() {
		t.Error("Elem of pointer should be settable")
	}

	// Test CanAddr
	if !v.CanAddr() {
		t.Error("Elem of pointer should be addressable")
	}
}

// Test Value.IsValid and IsNil
func TestValueIsValidIsNil(t *testing.T) {
	// Test IsValid
	v := reflect.ValueOf(42)
	if !v.IsValid() {
		t.Error("ValueOf(42) should be valid")
	}

	var zeroVal reflect.Value
	if zeroVal.IsValid() {
		t.Error("Zero Value should not be valid")
	}

	// Test IsNil
	var ptr *int
	v = reflect.ValueOf(ptr)
	if !v.IsNil() {
		t.Error("nil pointer Value should be nil")
	}

	i := 42
	v = reflect.ValueOf(&i)
	if v.IsNil() {
		t.Error("non-nil pointer Value should not be nil")
	}
}

// Test pointer operations
func TestPointerOperations(t *testing.T) {
	i := 42
	ptr := &i

	// Test Indirect
	v := reflect.ValueOf(ptr)
	indirect := reflect.Indirect(v)
	if indirect.Int() != 42 {
		t.Errorf("Indirect should give 42, got %d", indirect.Int())
	}

	// Test Elem
	elem := v.Elem()
	if elem.Int() != 42 {
		t.Errorf("Elem should give 42, got %d", elem.Int())
	}
}

// Test Value.Interface
func TestValueInterface(t *testing.T) {
	i := 42
	v := reflect.ValueOf(i)

	iface := v.Interface()
	if iface.(int) != 42 {
		t.Errorf("Interface should be 42, got %v", iface)
	}
}

// Test Value.Addr and Pointer
func TestValueAddrPointer(t *testing.T) {
	i := 42
	v := reflect.ValueOf(&i).Elem()

	// Test Addr
	if !v.CanAddr() {
		t.Fatal("Value should be addressable")
	}
	addr := v.Addr()
	if addr.Kind() != reflect.Pointer {
		t.Errorf("Addr should return Pointer, got %v", addr.Kind())
	}

	// Test Pointer
	ptr := addr.Pointer()
	if ptr == 0 {
		t.Error("Pointer should not be zero")
	}

	// Test UnsafePointer
	unsafePtr := addr.UnsafePointer()
	if unsafePtr == nil {
		t.Error("UnsafePointer should not be nil")
	}
}

// Test Value.Bytes
func TestValueBytes(t *testing.T) {
	b := []byte{1, 2, 3}
	v := reflect.ValueOf(b)

	bytes := v.Bytes()
	if len(bytes) != 3 {
		t.Errorf("Bytes length should be 3, got %d", len(bytes))
	}
	if bytes[0] != 1 {
		t.Errorf("Bytes[0] should be 1, got %d", bytes[0])
	}
}

// Test Value.SetBytes
func TestValueSetBytes(t *testing.T) {
	var b []byte
	v := reflect.ValueOf(&b).Elem()

	v.SetBytes([]byte{1, 2, 3})
	if len(b) != 3 {
		t.Errorf("SetBytes: length should be 3, got %d", len(b))
	}
	if b[0] != 1 {
		t.Errorf("SetBytes: b[0] should be 1, got %d", b[0])
	}
}

// Test Value.Complex
func TestValueComplex(t *testing.T) {
	c := complex(1, 2)
	v := reflect.ValueOf(c)

	result := v.Complex()
	if result != 1+2i {
		t.Errorf("Complex should be (1+2i), got %v", result)
	}
}

// Test Value.Uint
func TestValueUint(t *testing.T) {
	var u uint = 42
	v := reflect.ValueOf(u)

	if v.Uint() != 42 {
		t.Errorf("Uint should be 42, got %d", v.Uint())
	}
}

// Test Value Can* methods
func TestValueCanMethods(t *testing.T) {
	// CanInt
	v := reflect.ValueOf(42)
	if !v.CanInt() {
		t.Error("int Value should satisfy CanInt")
	}

	// CanUint
	v = reflect.ValueOf(uint(42))
	if !v.CanUint() {
		t.Error("uint Value should satisfy CanUint")
	}

	// CanFloat
	v = reflect.ValueOf(3.14)
	if !v.CanFloat() {
		t.Error("float Value should satisfy CanFloat")
	}

	// CanComplex
	v = reflect.ValueOf(complex(1, 2))
	if !v.CanComplex() {
		t.Error("complex Value should satisfy CanComplex")
	}

	// CanInterface
	v = reflect.ValueOf(42)
	if !v.CanInterface() {
		t.Error("normal Value should satisfy CanInterface")
	}
}

// Test Value.Set
func TestValueSetGeneric(t *testing.T) {
	var i int = 10
	v := reflect.ValueOf(&i).Elem()

	newVal := reflect.ValueOf(42)
	v.Set(newVal)

	if i != 42 {
		t.Errorf("Set: expected 42, got %d", i)
	}
}

// Test Value.SetZero
func TestValueSetZero(t *testing.T) {
	i := 42
	v := reflect.ValueOf(&i).Elem()

	v.SetZero()
	if i != 0 {
		t.Errorf("SetZero: expected 0, got %d", i)
	}
}

// Test Value overflow methods
func TestValueOverflow(t *testing.T) {
	var i int8
	v := reflect.ValueOf(&i).Elem()

	// Test OverflowInt
	if !v.OverflowInt(128) {
		t.Error("128 should overflow int8")
	}
	if v.OverflowInt(127) {
		t.Error("127 should not overflow int8")
	}

	// Test OverflowUint
	var u uint8
	v = reflect.ValueOf(&u).Elem()
	if !v.OverflowUint(256) {
		t.Error("256 should overflow uint8")
	}

	// Test OverflowFloat
	var f float32
	v = reflect.ValueOf(&f).Elem()
	if !v.OverflowFloat(1e39) {
		t.Error("1e39 should overflow float32")
	}
}

// Test ValueError
func TestValueError(t *testing.T) {
	var ve *reflect.ValueError
	_ = ve

	// Trigger a ValueError
	defer func() {
		if r := recover(); r != nil {
			if ve, ok := r.(*reflect.ValueError); ok {
				if ve.Error() == "" {
					t.Fatal("ValueError.Error returned empty")
				}
			}
		}
	}()

	v := reflect.ValueOf("string")
	v.Int() // Should panic with ValueError
}

// Test Value.Convert
func TestValueConvert(t *testing.T) {
	var i int = 42
	v := reflect.ValueOf(i)

	// Convert int to int64
	int64Type := reflect.TypeOf(int64(0))
	converted := v.Convert(int64Type)
	if converted.Int() != 42 {
		t.Errorf("Convert to int64 should be 42, got %d", converted.Int())
	}
}

// Test Value.CanConvert
func TestValueCanConvert(t *testing.T) {
	var i int = 42
	v := reflect.ValueOf(i)

	int64Type := reflect.TypeOf(int64(0))
	if !v.CanConvert(int64Type) {
		t.Error("int should be convertible to int64")
	}

	// Note: In Go, int can be converted to string (e.g., string(65) = "A")
	// so we skip this check
}

// Test Value.IsZero
func TestValueIsZero(t *testing.T) {
	// Test zero int
	v := reflect.ValueOf(0)
	if !v.IsZero() {
		t.Error("Zero int should be zero")
	}

	// Test non-zero int
	v = reflect.ValueOf(42)
	if v.IsZero() {
		t.Error("Non-zero int should not be zero")
	}

	// Test empty string
	v = reflect.ValueOf("")
	if !v.IsZero() {
		t.Error("Empty string should be zero")
	}
}

// Test Value.Comparable and Equal
func TestValueComparableEqual(t *testing.T) {
	v1 := reflect.ValueOf(42)
	v2 := reflect.ValueOf(42)
	v3 := reflect.ValueOf(43)

	// Test Comparable
	if !v1.Comparable() {
		t.Error("int Value should be comparable")
	}

	// Test Equal
	if !v1.Equal(v2) {
		t.Error("Equal values should be equal")
	}
	if v1.Equal(v3) {
		t.Error("Different values should not be equal")
	}
}

// Test Value.OverflowComplex
func TestValueOverflowComplex(t *testing.T) {
	var c complex64
	v := reflect.ValueOf(&c).Elem()

	// Test with large complex number
	large := complex(1e39, 1e39)
	if !v.OverflowComplex(large) {
		t.Error("Large complex should overflow complex64")
	}
}

// Test Value.UnsafeAddr
func TestValueUnsafeAddr(t *testing.T) {
	i := 42
	v := reflect.ValueOf(&i).Elem()

	if !v.CanAddr() {
		t.Fatal("Value should be addressable")
	}

	addr := v.UnsafeAddr()
	if addr == 0 {
		t.Error("UnsafeAddr should not be zero")
	}
}

// Test Value.InterfaceData (deprecated but still exists)
func TestValueInterfaceData(t *testing.T) {
	var i interface{} = 42
	v := reflect.ValueOf(&i).Elem()

	// InterfaceData is deprecated but we test it exists
	data := v.InterfaceData()
	if data[0] == 0 && data[1] == 0 {
		t.Fatal("InterfaceData returned zero words")
	}
}

// Test Value.SetPointer
func TestValueSetPointer(t *testing.T) {
	var p unsafe.Pointer
	v := reflect.ValueOf(&p).Elem()

	i := 42
	v.SetPointer(unsafe.Pointer(&i))

	if p == nil {
		t.Error("SetPointer should set non-nil pointer")
	}
}

// Test Value.FieldByIndexErr
func TestValueFieldByIndexErr(t *testing.T) {
	type Inner struct {
		Value int
	}
	type Outer struct {
		Inner
	}

	o := Outer{Inner{42}}
	v := reflect.ValueOf(o)

	// Valid index path
	field, err := v.FieldByIndexErr([]int{0, 0})
	if err != nil {
		t.Fatalf("FieldByIndexErr should not error on valid path: %v", err)
	}
	if field.Int() != 42 {
		t.Errorf("FieldByIndexErr should get 42, got %d", field.Int())
	}

	// Note: Testing error case would cause panic in current Go version
	// FieldByIndexErr exists but still panics on invalid index
}

// Test Value.Seq and Seq2 (Go 1.24+ iterators)
func TestValueSeq(t *testing.T) {
	// Seq returns an iterator for array/slice
	s := []int{1, 2, 3}
	v := reflect.ValueOf(s)

	// Just check that Seq method exists
	_ = v.Seq

	// Seq2 for maps
	m := map[string]int{"a": 1}
	v2 := reflect.ValueOf(m)
	_ = v2.Seq2
}
