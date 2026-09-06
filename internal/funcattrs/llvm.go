package funcattrs

import (
	"encoding/json"
	"fmt"
	"go/types"

	"github.com/xgo-dev/llvm"
)

// Metadata retains logical contracts through function recreation and bitcode.
// LLVM does not interpret this string; Apply and Remap materialize actual facts.
const Metadata = "llgo.source.attributes"

// LLVM 22 MemoryEffects: two ModRef bits per location, ArgMem first. CaptureInfo
// uses the low four bits for the return channel. Keep encodings in the backend.
const memoryRead = 0x555

func Apply(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment, intBits int) error {
	if len(attrs) == 0 {
		return nil
	}
	if err := Validate(attrs, sig, intBits, false); err != nil {
		return err
	}
	for _, a := range attrs {
		index := -1
		switch a.Target.Scope {
		case Result:
			index = 0
		case Receiver:
			index = 1 + environment
		case Parameter:
			index = 1 + environment + a.Target.Index
			if sig.Recv() != nil {
				index++
			}
		}
		name, value := a.Name, uint64(0)
		switch name {
		case "memory":
			switch a.Args {
			case "read":
				value = memoryRead
			case "argmem:read":
				value = 1
			case "argmem:readwrite":
				value = 3
			case "read,argmem:readwrite":
				value = memoryRead | 3
			}
		case "captures":
			if a.Args != "none" {
				value = 0xf
			}
		case "range", "nonnegative":
			bits, values, full, err := IntegerRange(a, valueType(sig, a.Target), intBits)
			if err != nil {
				return err
			}
			if !full {
				fn.AddAttributeAtIndex(index, ctx.CreateConstantRangeAttribute(llvm.AttributeKindID("range"), bits, values[:1], values[1:]))
			}
			continue
		case "returned":
			if fn.GlobalValueType().ReturnType() != fn.Param(index-1).Type() || !scalar(fn.GlobalValueType().ReturnType()) {
				return a.Error("returned requires an unchanged scalar ABI value in v1")
			}
		}
		fn.AddAttributeAtIndex(index, ctx.CreateEnumAttribute(llvm.AttributeKindID(name), value))
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	fn.AddFunctionAttr(ctx.CreateStringAttribute(Metadata, string(data)))
	return nil
}

func scalar(t llvm.Type) bool {
	switch t.TypeKind() {
	case llvm.PointerTypeKind, llvm.IntegerTypeKind, llvm.FloatTypeKind, llvm.DoubleTypeKind:
		return true
	}
	return false
}

// Representation describes where a logical ABI value goes. A source-level
// Target remains separate from this physical mapping. Later passes can describe
// fragments, field paths and storage instead of overloading a parameter index.
type Representation uint8

const (
	Direct Representation = iota
	Omitted
	Fragments
	Indirect
)

type ABIValue struct {
	Kind      Representation
	Indices   []int // LLVM attribute indices: 0 = return, 1.. = arguments
	FieldPath []int
}

type ABIMapping struct {
	Result ABIValue
	Params []ABIValue // indexed by the previous LLVM parameter position
}

func DirectValue(index int) ABIValue { return ABIValue{Kind: Direct, Indices: []int{index}} }

var valueAttributes = []string{"nonnull", "readonly", "writeonly", "captures", "returned", "range"}

// RemapFunction preserves the v1 facts whenever an ABI recreates a function,
// even when only an unrelated aggregate argument changes the signature.
func RemapFunction(from, to llvm.Value, m ABIMapping) error {
	strict := !from.GetStringAttributeAtIndex(-1, Metadata).IsNil()
	failure := func(err error) error {
		if err == nil {
			return nil
		}
		if strict {
			var attrs []Attribute
			if json.Unmarshal([]byte(from.GetStringAttributeAtIndex(-1, Metadata).GetStringValue()), &attrs) == nil && len(attrs) != 0 {
				return attrs[0].Error("%s: %v", to.Name(), err)
			}
		}
		return err
	}
	if strict && m.Result.Kind == Indirect {
		if attr := from.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")); !attr.IsNil() && attr.GetEnumValue()&2 == 0 {
			return failure(fmt.Errorf("memory contract does not support an indirect ABI result in v1"))
		}
	}
	return failure(remap(to.Name(), strict, m, from.GetEnumAttributeAtIndex, to.AddAttributeAtIndex))
}

// RemapCall also handles explicit call-site attributes produced by other
// lowering paths. Direct-call contracts remain available on the declaration.
func RemapCall(from, to llvm.Value, m ABIMapping) error {
	strict := !from.GetCallSiteStringAttribute(-1, Metadata).IsNil()
	return remap("call", strict, m, from.GetCallSiteEnumAttribute, to.AddCallSiteAttribute)
}

func remap(name string, strict bool, m ABIMapping, get func(int, uint) llvm.Attribute, add func(int, llvm.Attribute)) error {
	values := append([]ABIValue{m.Result}, m.Params...)
	for old, v := range values {
		for _, a := range valueAttributes {
			attr := get(old, llvm.AttributeKindID(a))
			if attr.IsNil() {
				continue
			}
			if v.Kind != Direct || len(v.Indices) != 1 || a == "returned" && m.Result.Kind != Direct {
				if strict {
					return fmt.Errorf("llgo:attribute: %s: %s on ABI value %d is not supported after this transformation in v1", name, a, old)
				}
				continue
			}
			add(v.Indices[0], attr)
		}
	}
	return nil
}
