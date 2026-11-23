package nativerepr

import (
	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/native"
	"testing"
)

//go:noinline
func checkLocalDefinition[T any](value T) bool {
	type Copy native.Record
	frame := struct {
		Value native.Record
		Guard uint64
	}{Value: native.Record{Tail: 7}, Guard: 123456}
	type Alias = Copy
	type ThroughAlias Alias
	p := (*ThroughAlias)(&frame.Value)
	p.Tail = 99
	return frame.Value.Tail == 99 && frame.Guard == 123456
}

func TestNativeGenericLocalDefinition(t *testing.T) {
	if !checkLocalDefinition(1) || !checkLocalDefinition("different instantiation") {
		t.Fatal("instantiated local definition lost native storage")
	}
}

//llgo:cdecl
type localRoot struct {
	Fn   func() uint64
	Tail uint64
}

//go:noinline
func checkLocalShadow[T any](value T) bool {
	type localRoot struct {
		Fn   func() uint64
		Tail uint64
	}
	secret := uint64(40)
	v := localRoot{Fn: func() uint64 { secret++; return secret }, Tail: 1}
	return v.Fn()+v.Tail == 42
}

func TestNativeGenericLocalShadow(t *testing.T) {
	if !checkLocalShadow(1) || !checkLocalShadow("different instantiation") {
		t.Fatal("instantiated local shadow inherited the package native directive")
	}
}
