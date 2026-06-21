package gotest

import "testing"

type nilPointerInterfaceLarge [1 << 21]byte

type nilPointerInterfaceZero struct{}

type nilPointerInterfaceLargeStruct struct {
	data [1 << 21]byte
}

type nilPointerInterfaceLargeField struct {
	pad   [1 << 21]byte
	value nilPointerInterfaceLargeStruct
}

var nilPointerInterfaceSink any

func expectNilPointerInterfacePanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil pointer dereference panic")
		}
	}()
	f()
}

func TestNilPointerLargeArrayToInterfacePanics(t *testing.T) {
	expectNilPointerInterfacePanic(t, func() {
		var p *nilPointerInterfaceLarge
		nilPointerInterfaceSink = *p
	})
}

func TestNilPointerZeroSizedValueToInterfacePanics(t *testing.T) {
	expectNilPointerInterfacePanic(t, func() {
		var p *nilPointerInterfaceZero
		nilPointerInterfaceSink = *p
	})
}

func TestNilPointerLargeStructToInterfacePanics(t *testing.T) {
	expectNilPointerInterfacePanic(t, func() {
		var p *nilPointerInterfaceLargeStruct
		nilPointerInterfaceSink = *p
	})
}

func TestNilPointerLargeValueStandaloneDerefPanics(t *testing.T) {
	expectNilPointerInterfacePanic(t, func() {
		var p *nilPointerInterfaceLarge
		_ = *p
	})
}

func TestNilPointerLargeFieldToInterfacePanics(t *testing.T) {
	expectNilPointerInterfacePanic(t, func() {
		var p *nilPointerInterfaceLargeField
		nilPointerInterfaceSink = p.value
	})
}

func TestNilPointerLargeValueToInterfacePanics(t *testing.T) {
	expectNilPointerInterfacePanic(t, func() {
		var p *nilPointerInterfaceLarge
		nilPointerInterfaceSink = *p
	})
}
