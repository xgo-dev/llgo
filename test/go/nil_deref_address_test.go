package gotest

import (
	"fmt"
	"strings"
	"testing"
)

type nilDerefAddressStruct struct {
	i int
	x [2]int
}

var nilDerefAddressSink any
var nilDerefSlicePointer *[]byte
var nilDerefStructPointer *nilDerefAddressStruct
var nilDerefLargePointer *struct {
	pad   [8192]byte
	value int
}

func TestNilArrayPointerSlicesPanic(t *testing.T) {
	var p *[2]int
	var empty *[0]int
	var zeroSize *[2]struct{}
	for _, tt := range []struct {
		name string
		f    func()
	}{
		{"full", func() { nilDerefAddressSink = p[:] }},
		{"empty", func() { nilDerefAddressSink = p[:0] }},
		{"bounded", func() { nilDerefAddressSink = p[1:2] }},
		{"three indices", func() { nilDerefAddressSink = p[0:0:0] }},
		{"zero length", func() { nilDerefAddressSink = empty[:] }},
		{"zero size", func() { nilDerefAddressSink = zeroSize[:] }},
		{"unused", func() { _ = p[:] }},
	} {
		t.Run(tt.name, func(t *testing.T) { expectNilDerefAddressPanic(t, tt.f) })
	}
	// The zero-length result must retain a valid non-nil array base.
	a := [2]int{3, 4}
	if s := a[:0]; s == nil || cap(s) != 2 {
		t.Fatalf("non-nil empty array slice = %v", s)
	}
	var s []int
	if s[:] != nil {
		t.Fatal("slicing a nil slice must remain valid")
	}
}

func TestNilGlobalPointerLoadsPanic(t *testing.T) {
	for _, tt := range []struct {
		name string
		f    func()
	}{
		{"slice header", func() { println(*nilDerefSlicePointer) }},
		{"large field", func() { println(nilDerefLargePointer.value) }},
		{"large aggregate", func() { nilDerefAddressSink = *nilDerefLargePointer }},
		{"field array address", func() { nilDerefAddressSink = &nilDerefStructPointer.x[0] }},
		{"field array value", func() { nilDerefAddressSink = nilDerefStructPointer.x[0] }},
		{"field array slice", func() { nilDerefAddressSink = nilDerefStructPointer.x[:] }},
	} {
		t.Run(tt.name, func(t *testing.T) { expectNilDerefAddressPanic(t, tt.f) })
	}
}

func TestNilFieldArrayAssignmentOrder(t *testing.T) {
	var order []string
	index := func() int { order = append(order, "index"); return 0 }
	rhs := func() int { order = append(order, "rhs"); return 1 }
	expectNilDerefAddressPanic(t, func() { nilDerefStructPointer.x[index()] = rhs() })
	if got := strings.Join(order, ","); got != "index,rhs" {
		t.Fatalf("assignment evaluation order = %q", got)
	}
	order = nil
	expectNilDerefAddressPanic(t, func() { nilDerefAddressSink = &nilDerefStructPointer.x[index()] })
	if got := strings.Join(order, ","); got != "index" {
		t.Fatalf("address evaluation order = %q", got)
	}
}

func TestNilDerefAddressOperationsPanic(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "address of dereference",
			f: func() {
				var p *int
				nilDerefAddressSink = &*p
			},
		},
		{
			name: "field address",
			f: func() {
				var p *nilDerefAddressStruct
				nilDerefAddressSink = &p.i
			},
		},
		{
			name: "array pointer element address",
			f: func() {
				var p *[2]int
				nilDerefAddressSink = &p[0]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func TestUnusedNilDerefOperationsPanic(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "direct pointer",
			f: func() {
				var p *int
				_ = *p
			},
		},
		{
			name: "pointer loaded from array",
			f: func() {
				var values [2]*int
				_ = *values[1]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func TestNilDerefPrintedCompositeLoadsPanic(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "slice pointer load",
			f: func() {
				var p *[]byte
				println(*p)
			},
		},
		{
			name: "string pointer load",
			f: func() {
				var p *string
				println(*p)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func TestNilDerefPrintedFieldLoadPanic(t *testing.T) {
	expectNilDerefAddressPanic(t, func() {
		var p *nilDerefAddressStruct
		println(p.i)
	})
}

func TestNilDerefInterfaceCopiesPanic(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "array pointer to interface",
			f: func() {
				var p *[4]int
				nilDerefAddressSink = *p
			},
		},
		{
			name: "struct pointer to interface",
			f: func() {
				var p *nilDerefAddressStruct
				nilDerefAddressSink = *p
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func expectNilDerefAddressPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("expected nil pointer dereference panic")
		}
		if got := nilDerefAddressPanicString(err); !strings.Contains(got, "nil pointer dereference") {
			t.Fatalf("panic = %q, want nil pointer dereference", got)
		}
	}()
	f()
}

func nilDerefAddressPanicString(v any) string {
	if err, ok := v.(interface{ Error() string }); ok {
		return err.Error()
	}
	return fmt.Sprint(v)
}
