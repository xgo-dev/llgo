package gotest

import (
	"fmt"
	"strings"
	"testing"
)

type nilDerefAddressStruct struct {
	i int
	x [2]int
	s string
}

type nilDerefEmbeddedReceiver struct{}

func (*nilDerefEmbeddedReceiver) value() int { return 0 }

type nilDerefEmbeddedOuter struct {
	padding int
	nilDerefEmbeddedReceiver
}

var nilDerefAddressSink any
var nilDerefAddressStructPtr *nilDerefAddressStruct
var nilDerefAddressSlicePtr *[]byte
var nilDerefAddressStringPtr *string
var nilDerefAddressArrayPtr *[4]int
var nilDerefEmbeddedOuterPtr *nilDerefEmbeddedOuter
var nilDerefEmbeddedReceiverPtr *nilDerefEmbeddedReceiver

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
		{
			name: "array field element address",
			f: func() {
				nilDerefAddressSink = &nilDerefAddressStructPtr.x[0]
			},
		},
		{
			name: "array field variable element address",
			f: func() {
				i := 1
				nilDerefAddressSink = &nilDerefAddressStructPtr.x[i]
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
				println(*nilDerefAddressSlicePtr)
			},
		},
		{
			name: "string pointer load",
			f: func() {
				println(*nilDerefAddressStringPtr)
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
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "basic field",
			f: func() {
				println(nilDerefAddressStructPtr.i)
			},
		},
		{
			name: "multiword field",
			f: func() {
				println(nilDerefAddressStructPtr.s)
			},
		},
		{
			name: "array field element",
			f: func() {
				i := 1
				println(nilDerefAddressStructPtr.x[i])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func TestNilDerefInterfaceCopiesPanic(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "array pointer to interface",
			f: func() {
				nilDerefAddressSink = *nilDerefAddressArrayPtr
			},
		},
		{
			name: "struct pointer to interface",
			f: func() {
				nilDerefAddressSink = *nilDerefAddressStructPtr
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func TestNilDerefPromotedPointerReceiverPanic(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "promoted method",
			f: func() {
				nilDerefAddressSink = nilDerefEmbeddedOuterPtr.value()
			},
		},
		{
			name: "explicit embedded field",
			f: func() {
				nilDerefAddressSink = nilDerefEmbeddedOuterPtr.nilDerefEmbeddedReceiver.value()
			},
		},
		{
			name: "promoted method value",
			f: func() {
				method := nilDerefEmbeddedOuterPtr.value
				nilDerefAddressSink = method()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectNilDerefAddressPanic(t, tt.f)
		})
	}
}

func TestNilPointerReceiverRemainsAllowed(t *testing.T) {
	if got := nilDerefEmbeddedReceiverPtr.value(); got != 0 {
		t.Fatalf("nil pointer receiver method = %d, want 0", got)
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
