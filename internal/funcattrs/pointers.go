package funcattrs

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

func pointerEffectName(attr Attribute) string {
	if attr.Name == "noalias" {
		return "noalias"
	}
	if attr.Name == "access" {
		switch attr.Args {
		case "none":
			return "readnone"
		case "read":
			return "readonly"
		case "write":
			return "writeonly"
		}
	}
	return ""
}

// ApplyPointerEffects runs after source validation and before ABI conversion.
func ApplyPointerEffects(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment int) error {
	for _, attr := range attrs {
		name := pointerEffectName(attr)
		if name == "" {
			continue
		}
		index := 1 + environment
		if attr.Target.Scope == Parameter {
			index += attr.Target.Index
			if sig.Recv() != nil {
				index++
			}
		}
		if index > fn.ParamsCount() || fn.Param(index-1).Type().TypeKind() != llvm.PointerTypeKind {
			return attr.Error("selected pointer has no LLVM parameter")
		}
		fn.AddAttributeAtIndex(index, ctx.CreateEnumAttribute(llvm.AttributeKindID(name), 0))
	}
	return nil
}

// CheckPointerEffects rejects generated runtime operations whose accesses cannot
// yet be reconciled with a retained source restriction. Native indices may have
// changed during ABI conversion; source positions remain owned by the backend.
func CheckPointerEffects(fn llvm.Value, attrs []Attribute, reason string) error {
	for _, attr := range attrs {
		name := pointerEffectName(attr)
		if name == "" {
			continue
		}
		for index := 1; index <= fn.ParamsCount(); index++ {
			if !fn.GetEnumAttributeAtIndex(index, llvm.AttributeKindID(name)).IsNil() {
				return attr.Error("%s cannot yet account for %s", attr.Name, reason)
			}
		}
	}
	return nil
}
