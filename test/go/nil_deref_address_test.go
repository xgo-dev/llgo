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
			name: "array pointer element address before bounds",
			f: func() {
				var p *[2]int
				i := 3
				nilDerefAddressSink = &p[i]
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
