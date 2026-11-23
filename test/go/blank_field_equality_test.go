package gotest

import (
	"reflect"
	"testing"
	"unsafe"
)

type blankFieldComparable struct {
	A uintptr
	_ uintptr
	B uintptr
}

type blankFieldLayout struct {
	A uintptr
	H uintptr
	B uintptr
}

func TestBlankFieldIgnoredInStructEquality(t *testing.T) {
	x := blankFieldComparable{A: 1, B: 2}
	y := blankFieldComparable{A: 1, B: 2}

	(*blankFieldLayout)(unsafe.Pointer(&x)).H = 3
	(*blankFieldLayout)(unsafe.Pointer(&y)).H = 9

	if x != y {
		t.Fatalf("blank fields should be ignored in equality: %#v != %#v", x, y)
	}

	// Reserve enough entries to exercise the hashed table. Inserting and
	// looking up the same key with different blank bytes must agree.
	m := make(map[blankFieldComparable]uint32, 64)
	for i := uintptr(3); i < 67; i++ {
		m[blankFieldComparable{A: i, B: 2}] = uint32(i)
	}
	m[x] = 99
	if value, ok := m[y]; !ok || value != 99 {
		t.Fatal("lookup hashed the blank field", value, ok)
	}
	reflected := reflect.ValueOf(m)
	reflected.SetMapIndex(reflect.ValueOf(y), reflect.ValueOf(uint32(100)))
	if got := m[x]; got != 100 || len(m) != 65 {
		t.Fatal("reflection hashed the blank field", got, len(m))
	}
	delete(m, y)
	if _, ok := m[x]; ok || len(m) != 64 {
		t.Fatal("deletion hashed the blank field")
	}
}

type paddedScalarMapKey struct {
	A uint8
	B uint16
	C uint32
	P *uint32
}

func TestPaddingIgnoredInScalarStructMapKeys(t *testing.T) {
	referent := uint32(1)
	x := paddedScalarMapKey{A: 2, B: 3, C: 4, P: &referent}
	y := x
	padding := unsafe.Offsetof(x.B) - 1
	*(*byte)(unsafe.Add(unsafe.Pointer(&x), padding)) = 0x55
	*(*byte)(unsafe.Add(unsafe.Pointer(&y), padding)) = 0xaa
	if x != y || any(x) != any(y) {
		t.Fatal("struct equality observed padding")
	}

	m := make(map[paddedScalarMapKey]uint32, 64)
	for i := uint32(5); i < 69; i++ {
		m[paddedScalarMapKey{A: x.A, B: x.B, C: i, P: x.P}] = i
	}
	m[x] = 99
	if value, ok := m[y]; !ok || value != 99 {
		t.Fatal("map lookup hashed padding", value, ok)
	}
	reflect.ValueOf(m).SetMapIndex(reflect.ValueOf(y), reflect.ValueOf(uint32(100)))
	if got := m[x]; got != 100 || len(m) != 65 {
		t.Fatal("reflection did not update the equal key", got, len(m))
	}
	delete(m, y)
	if _, ok := m[x]; ok || len(m) != 64 {
		t.Fatal("deletion did not remove the equal key")
	}
}
