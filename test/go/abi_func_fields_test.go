package gotest

import (
	"reflect"
	"testing"
)

type abiFastInput struct {
	Message string
	Flags   uint8
}

type abiFastOutput struct {
	Value int
}

type abiFastMethods = struct {
	Flags            uint64
	Size             func(abiFastInput) abiFastOutput
	Marshal          func(abiFastInput) abiFastOutput
	Unmarshal        func(abiFastInput) abiFastOutput
	Merge            func(abiFastInput) abiFastOutput
	CheckInitialized func(abiFastInput) abiFastOutput
	Equal            func(abiFastInput) abiFastOutput
}

type abiCoderMessageInfo struct {
	methods abiFastMethods
	Tail    int
}

type abiMessageInfo struct {
	Name string
	abiCoderMessageInfo
}

func (mi *abiMessageInfo) initFastMethods() {
	if mi.methods.Marshal == nil && mi.methods.Size == nil {
		mi.methods.Flags |= 1
		mi.methods.Marshal = mi.marshal
		mi.methods.Size = mi.size
	}
	if mi.methods.Unmarshal == nil {
		mi.methods.Flags |= 2
		mi.methods.Unmarshal = mi.unmarshal
	}
	if mi.methods.CheckInitialized == nil {
		mi.methods.CheckInitialized = mi.checkInitialized
	}
	if mi.methods.Merge == nil {
		mi.methods.Merge = mi.merge
	}
	if mi.methods.Equal == nil {
		mi.methods.Equal = abiEqual
	}
}

func (mi *abiMessageInfo) size(in abiFastInput) abiFastOutput {
	return abiFastOutput{Value: len(mi.Name) + int(in.Flags) + 10}
}

func (mi *abiMessageInfo) marshal(in abiFastInput) abiFastOutput {
	return abiFastOutput{Value: len(mi.Name) + len(in.Message) + 20}
}

func (mi *abiMessageInfo) unmarshal(in abiFastInput) abiFastOutput {
	return abiFastOutput{Value: len(mi.Name) + len(in.Message) + 30}
}

func (mi *abiMessageInfo) merge(in abiFastInput) abiFastOutput {
	return abiFastOutput{Value: len(mi.Name) + len(in.Message) + 40}
}

func (mi *abiMessageInfo) checkInitialized(in abiFastInput) abiFastOutput {
	return abiFastOutput{Value: len(mi.Name) + len(in.Message) + 50}
}

func abiEqual(in abiFastInput) abiFastOutput {
	return abiFastOutput{Value: len(in.Message) + 60}
}

func TestReflectABIDoesNotPolluteFunctionFieldLayout(t *testing.T) {
	// reflect.TypeOf forces llgo to emit ABI struct-field metadata for the
	// enclosing named struct before the fast-path function fields are assigned.
	typ := reflect.TypeOf(abiMessageInfo{})
	field, ok := typ.FieldByName("abiCoderMessageInfo")
	if !ok {
		t.Fatalf("embedded coder field not found in %v", typ)
	}
	if field.Offset == 0 {
		t.Fatalf("embedded coder field offset = 0, want after Name field")
	}

	mi := &abiMessageInfo{Name: "msg"}
	mi.initFastMethods()

	if got, want := mi.methods.Flags, uint64(3); got != want {
		t.Fatalf("method flags = %d, want %d", got, want)
	}
	cases := []struct {
		name string
		fn   func(abiFastInput) abiFastOutput
		want int
	}{
		{"Size", mi.methods.Size, 18},
		{"Marshal", mi.methods.Marshal, 26},
		{"Unmarshal", mi.methods.Unmarshal, 36},
		{"Merge", mi.methods.Merge, 46},
		{"CheckInitialized", mi.methods.CheckInitialized, 56},
		{"Equal", mi.methods.Equal, 63},
	}
	for _, tc := range cases {
		if tc.fn == nil {
			t.Fatalf("%s function is nil", tc.name)
		}
		if got := tc.fn(abiFastInput{Message: "abc", Flags: 5}).Value; got != tc.want {
			t.Fatalf("%s returned %d, want %d", tc.name, got, tc.want)
		}
	}
}
