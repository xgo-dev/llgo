package funcattrs

import (
	"encoding/json"
	"fmt"
	"go/types"
	"strings"

	"github.com/xgo-dev/llvm"
)

// AccessMode describes source accesses, independently of a backend's encoding.
// The two bits form a lattice: joining effects is a bitwise union.
type AccessMode uint8

const (
	AccessNone AccessMode = iota
	AccessRead
	AccessWrite
	AccessReadWrite
)

// MemoryEffects bounds observable state accesses, classified by source pointer
// root. Args does not include accesses through a pointer merely because it
// aliases an input: the access must derive from an entry pointer leaf. Other
// includes other program storage and external state. I/O, volatile/device
// access, and observations such as clocks or entropy which can change without
// a program write require Other=AccessReadWrite, modelling an observable event.
// Both memory(none) and memory(read) exclude these interactions.
type MemoryEffects struct {
	Args  AccessMode
	Other AccessMode
}

type EffectDisposition string

const (
	EffectNative       EffectDisposition = "native"
	EffectConservative EffectDisposition = "conservative"
)

// EffectLowering reports what the backend actually consumed. A retained source
// contract with no native carrier is explicitly conservative, not an optimized
// fact merely because it survived serialization.
type EffectLowering struct {
	Target        Target
	Name          string
	Disposition   EffectDisposition
	Reason        string `json:",omitempty"`
	PhysicalIndex int    `json:",omitempty"` // backend parameter coordinate, separate from Target
}

const EffectsMetadata = "llgo.source.effects.lowering.v1"

func ParseAccessMode(s string) (AccessMode, error) {
	switch strings.TrimSpace(s) {
	case "none":
		return AccessNone, nil
	case "read":
		return AccessRead, nil
	case "write":
		return AccessWrite, nil
	case "readwrite":
		return AccessReadWrite, nil
	}
	return 0, fmt.Errorf("unknown access mode %q", s)
}

// ParseMemoryEffects accepts a default mode and optional args/other overrides.
// An omitted default is none. Repeated locations/defaults are errors so the
// result never depends on the annotation's word order.
func ParseMemoryEffects(s string) (MemoryEffects, error) {
	var effects MemoryEffects
	var defaultMode AccessMode
	var args, other *AccessMode
	hasDefault := false
	for _, part := range strings.Split(s, ",") {
		location, mode, located := strings.Cut(part, ":")
		if !located {
			if hasDefault {
				return effects, fmt.Errorf("memory has multiple default modes")
			}
			value, err := ParseAccessMode(part)
			if err != nil {
				return effects, err
			}
			defaultMode, hasDefault = value, true
			continue
		}
		value, err := ParseAccessMode(mode)
		if err != nil {
			return effects, err
		}
		switch strings.TrimSpace(location) {
		case "args":
			if args != nil {
				return effects, fmt.Errorf("memory repeats args")
			}
			args = &value
		case "other":
			if other != nil {
				return effects, fmt.Errorf("memory repeats other")
			}
			other = &value
		default:
			return effects, fmt.Errorf("unknown memory location %q", strings.TrimSpace(location))
		}
	}
	effects.Args, effects.Other = defaultMode, defaultMode
	if args != nil {
		effects.Args = *args
	}
	if other != nil {
		effects.Other = *other
	}
	return effects, nil
}

// LLVM 22 has six locations with two ModRef bits each. Keep these physical
// encodings in the backend: they are not part of the source contract format.
const (
	nativeAllMemory      uint64 = 0xfff
	nativeArgumentMemory uint64 = 0x3
	nativeReturnCapture  uint64 = 0xf
	nativeAllCapture     uint64 = 0xff
)

// NativeMemoryEffects widens logical input accesses when a pointer leaf is
// carried in an aggregate rather than being a native pointer argument. This
// loses some alias precision, but never assigns a pointee's access permission
// to the byval container that happens to transport its pointer.
func NativeMemoryEffects(effects MemoryEffects, hiddenRoots bool) uint64 {
	other := effects.Other
	if hiddenRoots {
		other |= effects.Args
	}
	return (uint64(other)*0x555)&^nativeArgumentMemory | uint64(effects.Args)
}

// HasHiddenPointerRoots reports pointer leaves which native parameter
// attributes cannot describe. An ordinary scalar pointer is already a root.
func HasHiddenPointerRoots(fn llvm.Value) bool {
	for _, param := range fn.GlobalValueType().ParamTypes() {
		if param.TypeKind() != llvm.PointerTypeKind && containsPointer(param) {
			return true
		}
	}
	return false
}

func containsPointer(t llvm.Type) bool {
	switch t.TypeKind() {
	case llvm.PointerTypeKind:
		return true
	case llvm.StructTypeKind:
		for _, field := range t.StructElementTypes() {
			if containsPointer(field) {
				return true
			}
		}
	case llvm.ArrayTypeKind, llvm.VectorTypeKind:
		return containsPointer(t.ElementType())
	}
	return false
}

var parameterEffectAttributes = []string{"readnone", "readonly", "writeonly", "captures", "noalias"}

// RemapFunctionEffects must run after any blanket copying of native function
// attributes. ABI lowering adds effects even when no source contract exists.
func RemapFunctionEffects(from, to llvm.Value, mapping ABIMapping) {
	remapEffects(from.GlobalValueType(), to.Type().Context(), mapping,
		from.GetEnumAttributeAtIndex, to.AddAttributeAtIndex, to.RemoveEnumAttributeAtIndex)
	remapEffectDispositions(from, to, mapping)
}

func remapEffectDispositions(from, to llvm.Value, mapping ABIMapping) {
	attr := from.GetStringAttributeAtIndex(-1, EffectsMetadata)
	if attr.IsNil() {
		return
	}
	var lowering []EffectLowering
	if err := json.Unmarshal([]byte(attr.GetStringValue()), &lowering); err != nil {
		panic(err)
	}
	for i := range lowering {
		record := &lowering[i]
		old := record.PhysicalIndex
		if old <= 0 || old > len(mapping.Params) {
			continue
		}
		param := mapping.Params[old-1]
		record.PhysicalIndex = 0
		if param.Kind == Direct && len(param.Indices) == 1 {
			record.PhysicalIndex = param.Indices[0]
		} else {
			record.Disposition, record.Reason = EffectConservative, "ABI conversion has no native pointer parameter for the selected value"
		}
		if record.Name == "capture" && mapping.Result.Kind == Indirect {
			capture := from.GetEnumAttributeAtIndex(old, llvm.AttributeKindID("captures"))
			if !capture.IsNil() && capture.GetEnumValue()&nativeReturnCapture != 0 {
				record.Disposition, record.Reason = EffectConservative, "logical result capture becomes a store through the indirect result slot"
			}
		}
	}
	data, err := json.Marshal(lowering)
	if err != nil {
		panic(err)
	}
	to.AddFunctionAttr(to.Type().Context().CreateStringAttribute(EffectsMetadata, string(data)))
}

func RemapCallEffects(from, to llvm.Value, mapping ABIMapping) {
	ctx := to.Type().Context()
	remapEffects(from.CalledFunctionType(), to.Type().Context(), mapping,
		from.GetCallSiteEnumAttribute, to.AddCallSiteAttribute, func(index int, kind uint) {
			// The binding does not expose removal of a call-site attribute.
			// An explicit unconstrained capture replaces an earlier copied
			// restriction with exactly the same meaning as no attribute.
			to.AddCallSiteAttribute(index, ctx.CreateEnumAttribute(kind, nativeAllCapture))
		})
}

// ApplyEffects consumes the typed source model. Value facts and their native
// indices are handled separately by Apply and the boundary materializer.
func ApplyEffects(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment, intBits int) error {
	_ = intBits // Behavior contracts do not depend on the integer width.
	var lowering []EffectLowering
	for _, attr := range attrs {
		record := EffectLowering{Target: attr.Target, Name: attr.Name, Disposition: EffectNative}
		if attr.Target.Scope == Function {
			switch attr.Name {
			case "memory":
				effects := NativeMemoryEffects(attr.Memory, HasHiddenPointerRoots(fn))
				// A contract assume models an already-required condition, not
				// an observable storage access. LLVM's intrinsic-only control
				// dependency does not widen the enclosing memory contract.
				fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("memory"), effects))
				if HasHiddenPointerRoots(fn) && attr.Memory.Args&^attr.Memory.Other != 0 {
					record.Disposition, record.Reason = EffectConservative, "input pointer roots are carried inside aggregates"
				}
			case "cold", "noreturn":
				fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID(attr.Name), 0))
			default:
				continue
			}
			lowering = append(lowering, record)
			continue
		}
		if attr.Name != "access" && attr.Name != "capture" && attr.Name != "noalias" {
			continue
		}
		index := 1 + environment
		if attr.Target.Scope == Parameter {
			index += attr.Target.Index
			if sig.Recv() != nil {
				index++
			}
		}
		// A contract on an aggregate leaf cannot be attached to its carrier.
		// Keep the typed source fact; the native invocation summary is wider.
		if index > fn.ParamsCount() || fn.Param(index-1).Type().TypeKind() != llvm.PointerTypeKind {
			record.Disposition, record.Reason = EffectConservative, "selected pointer has no native pointer parameter"
			lowering = append(lowering, record)
			continue
		}
		name, value := "", uint64(0)
		if attr.Name == "noalias" {
			name = "noalias"
		} else if attr.Name == "access" {
			switch attr.Access {
			case AccessNone:
				name = "readnone"
			case AccessRead:
				name = "readonly"
			case AccessWrite:
				name = "writeonly"
			}
		} else {
			switch attr.Capture {
			case CaptureNone:
				name = "captures"
			case CaptureResults:
				name, value = "captures", nativeReturnCapture
			}
		}
		if name != "" {
			fn.AddAttributeAtIndex(index, ctx.CreateEnumAttribute(llvm.AttributeKindID(name), value))
		}
		record.PhysicalIndex = index
		lowering = append(lowering, record)
	}
	if len(lowering) != 0 {
		data, err := json.Marshal(lowering)
		if err != nil {
			return err
		}
		fn.AddFunctionAttr(ctx.CreateStringAttribute(EffectsMetadata, string(data)))
	}
	return nil
}

func markConservativeEffects(fn llvm.Value, reason string, names ...string) {
	attr := fn.GetStringAttributeAtIndex(-1, EffectsMetadata)
	if attr.IsNil() {
		return
	}
	var lowering []EffectLowering
	if err := json.Unmarshal([]byte(attr.GetStringValue()), &lowering); err != nil {
		panic(err)
	}
	for i := range lowering {
		for _, name := range names {
			if lowering[i].Name == name {
				lowering[i].Disposition, lowering[i].Reason = EffectConservative, reason
			}
		}
	}
	data, err := json.Marshal(lowering)
	if err != nil {
		panic(err)
	}
	fn.AddFunctionAttr(fn.Type().Context().CreateStringAttribute(EffectsMetadata, string(data)))
}

func remapEffects(oldType llvm.Type, ctx llvm.Context, mapping ABIMapping,
	get func(int, uint) llvm.Attribute, add func(int, llvm.Attribute), remove func(int, uint),
) {
	memoryKind := llvm.AttributeKindID("memory")
	memory := get(-1, memoryKind)
	if !memory.IsNil() {
		effects := memory.GetEnumValue()
		oldParams := oldType.ParamTypes()
		for i, param := range mapping.Params {
			if param.Kind == Indirect {
				// The callee reads the newly introduced transfer slot.
				effects |= uint64(AccessRead)
			}
			if i < len(oldParams) && oldParams[i].TypeKind() == llvm.PointerTypeKind &&
				(param.Kind != Direct || len(param.Indices) != 1) {
				// A formerly direct pointer root can no longer be classified as
				// physical argument memory. Preserve its possible accesses in
				// every physical region, not on the transport pointer alone.
				effects |= (memory.GetEnumValue() & nativeArgumentMemory) * 0x555
			}
		}
		if mapping.Result.Kind == Indirect {
			effects |= uint64(AccessWrite)
		}
		add(-1, ctx.CreateEnumAttribute(memoryKind, effects))
	}
	for i, param := range mapping.Params {
		if param.Kind != Direct || len(param.Indices) != 1 {
			continue
		}
		index := param.Indices[0]
		for _, name := range parameterEffectAttributes {
			kind := llvm.AttributeKindID(name)
			attr := get(i+1, kind)
			if attr.IsNil() {
				continue
			}
			if name == "captures" && mapping.Result.Kind == Indirect &&
				attr.GetEnumValue()&nativeReturnCapture != 0 {
				// Logical result capture becomes a store through sret. LLVM's
				// native return channel does not include that store.
				remove(index, kind)
				continue
			}
			add(index, attr)
		}
	}
}

func addFunctionMemoryEffects(fn llvm.Value, extra uint64) {
	kind := llvm.AttributeKindID("memory")
	attr := fn.GetEnumAttributeAtIndex(-1, kind)
	if !attr.IsNil() {
		fn.AddFunctionAttr(fn.Type().Context().CreateEnumAttribute(kind, attr.GetEnumValue()|extra))
	}
}

// WidenForGCRootPublication accounts for roots stored into a published chain.
// It does not disable the required publication or reinterpret it as a private
// store. The ordinary root prologue itself does not free, sync, or unwind.
func WidenForGCRootPublication(fn llvm.Value) {
	addFunctionMemoryEffects(fn, nativeAllMemory&^nativeArgumentMemory)
	for i := 1; i <= fn.ParamsCount(); i++ {
		fn.RemoveEnumAttributeAtIndex(i, llvm.AttributeKindID("captures"))
	}
	markConservativeEffects(fn, "compiler-generated GC root publication", "memory", "capture")
}

// WidenForUnknownInstrumentation preserves hints and no-normal-return while
// dropping restrictions that an inserted runtime call can invalidate. Apply
// the same policy to imported declarations; widening only a definition after
// its callers have been optimized is not sufficient.
func WidenForUnknownInstrumentation(fn llvm.Value) {
	for _, name := range []string{"memory", "nofree", "nosync", "nounwind", "willreturn"} {
		fn.RemoveEnumAttributeAtIndex(-1, llvm.AttributeKindID(name))
	}
	for i := 1; i <= fn.ParamsCount(); i++ {
		for _, name := range parameterEffectAttributes {
			fn.RemoveEnumAttributeAtIndex(i, llvm.AttributeKindID(name))
		}
	}
	markConservativeEffects(fn, "target mode permits compiler-generated runtime instrumentation", "memory", "nofree", "nosync", "nounwind", "willreturn", "access", "capture", "noalias")
}

// CheckInstrumentation rejects an unmodelled generated operation only when a
// native restriction still relies on the source promise. Target-wide widening
// can make the operation safe without changing the source contract. Unlike a
// local relaxation, a diagnostic cannot leave previously compiled callers with
// a stale, stronger declaration.
func CheckInstrumentation(fn llvm.Value, reason string, names ...string) error {
	metadata := fn.GetStringAttributeAtIndex(-1, Metadata)
	if metadata.IsNil() {
		return nil
	}
	var attrs []Attribute
	if err := json.Unmarshal([]byte(metadata.GetStringValue()), &attrs); err != nil {
		return err
	}
	for _, attr := range attrs {
		for _, name := range names {
			if name == "captures" {
				name = "capture"
			}
			if attr.Name == name && HasNativeEffectRestriction(fn, name) {
				return attr.Error("%s cannot yet account for %s", name, reason)
			}
		}
	}
	return nil
}

func HasNativeEffectRestriction(fn llvm.Value, name string) bool {
	if name == "capture" || name == "access" || name == "noalias" {
		attributes := []string{"captures"}
		if name == "noalias" {
			attributes = []string{"noalias"}
		}
		if name == "access" {
			attributes = []string{"readonly", "writeonly", "readnone"}
		}
		for i := 1; i <= fn.ParamsCount(); i++ {
			for _, attribute := range attributes {
				attr := fn.GetEnumAttributeAtIndex(i, llvm.AttributeKindID(attribute))
				if !attr.IsNil() && (attribute != "captures" || attr.GetEnumValue() != nativeAllCapture) {
					return true
				}
			}
		}
		return false
	}
	attr := fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name))
	return !attr.IsNil() && (name != "memory" || attr.GetEnumValue() != nativeAllMemory)
}
