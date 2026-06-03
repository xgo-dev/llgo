package gotest

import (
	"reflect"
	"testing"
	"unsafe"
)

type reflectCallWordArgHolder struct {
	P *int
	C chan int
	M map[string]int
	U unsafe.Pointer
}

func reflectCallWordArgTake(p *int, c chan int, m map[string]int, u unsafe.Pointer) (int, int, int, int) {
	return *p, cap(c), m["x"], *(*int)(u)
}

func TestReflectCallIndirectWordArgs(t *testing.T) {
	x := 7
	h := reflectCallWordArgHolder{
		P: &x,
		C: make(chan int, 3),
		M: map[string]int{"x": 11},
		U: unsafe.Pointer(&x),
	}
	v := reflect.ValueOf(h)
	out := reflect.ValueOf(reflectCallWordArgTake).Call([]reflect.Value{
		v.Field(0),
		v.Field(1),
		v.Field(2),
		v.Field(3),
	})

	got := []int{int(out[0].Int()), int(out[1].Int()), int(out[2].Int()), int(out[3].Int())}
	want := []int{7, 3, 11, 7}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("result[%d] = %d, want %d; all results: %v", i, got[i], want[i], got)
		}
	}
}
